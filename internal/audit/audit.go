package audit

import (
	"encoding/json"
	"io"
	"time"
)

const SchemaVersion = "mcp-data-gateway.audit.v1"

type Event struct {
	SchemaVersion string    `json:"schema_version"`
	EventID       string    `json:"event_id"`
	Timestamp     time.Time `json:"timestamp"`
	RequestID     string    `json:"request_id"`
	Actor         Actor     `json:"actor"`
	Client        Client    `json:"client"`
	Decision      Decision  `json:"decision"`
	Result        Result    `json:"result"`
}

type Actor struct {
	Subject string   `json:"subject"`
	Groups  []string `json:"groups,omitempty"`
}

type Client struct {
	Name     string `json:"name"`
	EdgeMode string `json:"edge_mode"`
}

type Decision struct {
	Allowed  bool   `json:"allowed"`
	ToolPack string `json:"tool_pack,omitempty"`
	Tool     string `json:"tool,omitempty"`
}

type Result struct {
	Status           string `json:"status"`
	RedactionApplied bool   `json:"redaction_applied"`
}

func Write(w io.Writer, event Event) error {
	if event.SchemaVersion == "" {
		event.SchemaVersion = SchemaVersion
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	enc := json.NewEncoder(w)
	return enc.Encode(event)
}
