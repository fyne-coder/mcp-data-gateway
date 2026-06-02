package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/fyne-coder/mcp-data-gateway/internal/auth"
	"github.com/fyne-coder/mcp-data-gateway/internal/buildinfo"
	"github.com/fyne-coder/mcp-data-gateway/internal/config"
	"github.com/fyne-coder/mcp-data-gateway/internal/policy"
	"github.com/fyne-coder/mcp-data-gateway/internal/postgres"
	"github.com/fyne-coder/mcp-data-gateway/internal/resultshape"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type invocationContextKey struct{}

// Invocation carries authenticated actor context for MCP tool handlers.
type Invocation struct {
	Actor auth.Actor
}

// WithInvocation stores actor context on the request.
func WithInvocation(ctx context.Context, inv Invocation) context.Context {
	return context.WithValue(ctx, invocationContextKey{}, inv)
}

func InvocationFromContext(ctx context.Context) (Invocation, bool) {
	inv, ok := ctx.Value(invocationContextKey{}).(Invocation)
	return inv, ok
}

// Deps configures the MCP server factory.
type Deps struct {
	Policy   policy.Engine
	Postgres config.PostgresConfig
	Querier  postgres.Querier
	Shaper   resultshape.Shaper
}

// StreamableHTTPHandler returns an SDK-backed streamable HTTP handler for POST /mcp.
func StreamableHTTPHandler(deps Deps) http.Handler {
	if deps.Shaper == nil {
		deps.Shaper = resultshape.Passthrough{}
	}
	return mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return newServer(r, deps) },
		&mcp.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		},
	)
}

func newServer(_ *http.Request, deps Deps) *mcp.Server {
	srv := mcp.NewServer(
		&mcp.Implementation{Name: "mcp-data-gateway", Version: buildinfo.Version},
		&mcp.ServerOptions{HasTools: true},
	)
	registerPostgresSelect(srv, deps)
	return srv
}

func registerPostgresSelect(srv *mcp.Server, deps Deps) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        postgres.ToolName,
		Description: "Read-only select against allowlisted Postgres tables and columns.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input postgres.SelectInput) (*mcp.CallToolResult, postgres.SelectOutput, error) {
		inv, ok := InvocationFromContext(ctx)
		if !ok || inv.Actor.Subject == "" {
			return nil, postgres.SelectOutput{}, errors.New("missing invocation context")
		}

		decision := deps.Policy.Decide(policy.Request{
			Actor: policy.Actor{Subject: inv.Actor.Subject, Groups: inv.Actor.Groups},
			Tool:  policy.ToolPostgresSelect,
		})
		if !decision.Allowed {
			return nil, postgres.SelectOutput{}, fmt.Errorf("tool pack not allowed: %s", decision.Reason)
		}

		selector := postgres.Selector{Allowlists: deps.Postgres, Querier: deps.Querier}
		out, err := selector.Select(ctx, decision.ToolPack, input)
		if err != nil {
			return nil, postgres.SelectOutput{}, err
		}

		shaped, err := deps.Shaper.ShapeRows(out.Rows)
		if err != nil {
			return nil, postgres.SelectOutput{}, err
		}
		out.Rows = shaped
		out.RowCount = len(shaped)
		return nil, out, nil
	})
}
