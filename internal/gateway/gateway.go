package gateway

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fyne-coder/mcp-data-gateway/internal/audit"
	"github.com/fyne-coder/mcp-data-gateway/internal/auth"
	"github.com/fyne-coder/mcp-data-gateway/internal/config"
	"github.com/fyne-coder/mcp-data-gateway/internal/mcpserver"
	"github.com/fyne-coder/mcp-data-gateway/internal/policy"
	"github.com/fyne-coder/mcp-data-gateway/internal/postgres"
	"github.com/fyne-coder/mcp-data-gateway/internal/resultshape"
)

type Server struct {
	config     config.Config
	verifier   auth.Verifier
	audit      audit.Sink
	policy     policy.Engine
	querier    postgres.Querier
	mcpHandler http.Handler
}

func New(cfg config.Config) (Server, error) {
	verifier, err := auth.NewOIDCVerifier(context.Background(), cfg.Auth)
	if err != nil {
		return Server{}, err
	}
	sink, err := audit.NewSink(cfg.Audit)
	if err != nil {
		return Server{}, err
	}
	querier := postgres.Querier(postgres.UnconfiguredQuerier{DSNEnv: cfg.Postgres.DSNEnv})
	if dsn := os.Getenv(cfg.Postgres.DSNEnv); dsn != "" {
		pool, err := postgres.NewPoolFromEnv(cfg.Postgres.DSNEnv)
		if err != nil {
			return Server{}, fmt.Errorf("failed to initialize postgres pool from env %s", cfg.Postgres.DSNEnv)
		}
		querier = postgres.PoolQuerier{Pool: pool}
	}
	return NewWithDeps(cfg, verifier, sink, querier), nil
}

func NewWithVerifierAndSink(cfg config.Config, verifier auth.Verifier, sink audit.Sink) Server {
	return NewWithDeps(cfg, verifier, sink, nil)
}

// NewWithDeps constructs a gateway server. When querier is nil, postgres_select
// fails safely with an unconfigured Postgres error.
func NewWithDeps(cfg config.Config, verifier auth.Verifier, sink audit.Sink, querier postgres.Querier) Server {
	if querier == nil {
		querier = postgres.UnconfiguredQuerier{DSNEnv: cfg.Postgres.DSNEnv}
	}
	policyEngine := policy.Engine{
		DefaultToolPack: cfg.Policy.DefaultToolPack,
		GroupToolPacks:  cfg.Policy.GroupToolPacks,
	}
	return Server{
		config:   cfg,
		verifier: verifier,
		audit:    sink,
		policy:   policyEngine,
		querier:  querier,
		mcpHandler: mcpserver.StreamableHTTPHandler(mcpserver.Deps{
			Policy:   policyEngine,
			Postgres: cfg.Postgres,
			Querier:  querier,
			Shaper:   resultshape.Passthrough{},
		}),
	}
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
	mux.HandleFunc("/mcp", s.handleMCP)
	return mux
}

func (s Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requestID := audit.RequestID(r)
	eventID := audit.NewEventID()
	w.Header().Set("X-Request-Id", requestID)

	emit := func(actor audit.Actor, allowed bool, status string) {
		event := audit.MCPEvent(
			requestID,
			eventID,
			s.config.Edge.Mode,
			s.config.Policy.DefaultToolPack,
			actor,
			allowed,
			status,
		)
		_ = s.audit.Emit(event)
	}

	rawToken, err := auth.ParseBearer(r.Header.Get("Authorization"))
	if err != nil {
		emit(audit.Actor{}, false, audit.ResultUnauthorized)
		http.Error(w, "missing or invalid authorization", auth.HTTPStatus(err))
		return
	}

	actor, err := s.verifier.Verify(r.Context(), rawToken)
	if err != nil {
		emit(audit.Actor{}, false, audit.ResultUnauthorized)
		http.Error(w, "invalid bearer token", auth.HTTPStatus(err))
		return
	}

	auditActor := audit.Actor{Subject: actor.Subject, Groups: actor.Groups}
	if err := auth.Authorize(actor, s.config.Auth.RequiredGroups); err != nil {
		emit(auditActor, false, audit.ResultForbidden)
		http.Error(w, "insufficient group membership", auth.HTTPStatus(err))
		return
	}

	recorder := &responseStatusRecorder{ResponseWriter: w}
	ctx := mcpserver.WithInvocation(r.Context(), mcpserver.Invocation{Actor: actor})
	s.mcpHandler.ServeHTTP(recorder, r.WithContext(ctx))
	emit(auditActor, true, audit.MCPResultStatus(recorder.statusCode(), recorder.body.Bytes()))
}

type responseStatusRecorder struct {
	http.ResponseWriter
	code int
	body bytes.Buffer
}

func (r *responseStatusRecorder) WriteHeader(status int) {
	r.code = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseStatusRecorder) Write(b []byte) (int, error) {
	if r.code == 0 {
		r.code = http.StatusOK
	}
	_, _ = r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseStatusRecorder) statusCode() int {
	if r.code == 0 {
		return http.StatusOK
	}
	return r.code
}

func (s Server) ListenAndServe() error {
	server := &http.Server{
		Addr:              s.config.Server.Listen,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return server.ListenAndServe()
}

func (s Server) Summary() string {
	return fmt.Sprintf("listen=%s edge=%s issuer=%s", s.config.Server.Listen, s.config.Edge.Mode, s.config.Auth.Issuer)
}
