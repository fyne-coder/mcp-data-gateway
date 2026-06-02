package audit

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/fyne-coder/mcp-data-gateway/internal/config"
)

// Sink emits newline-delimited audit events.
type Sink interface {
	Emit(event Event) error
}

// WriterSink writes events to an io.Writer.
type WriterSink struct {
	w io.Writer
}

func (s WriterSink) Emit(event Event) error {
	return Write(s.w, event)
}

// DiscardSink drops events.
type DiscardSink struct{}

func (DiscardSink) Emit(Event) error { return nil }

// MemorySink records events for tests.
type MemorySink struct {
	mu     sync.Mutex
	Events []Event
}

func (s *MemorySink) Emit(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, event)
	return nil
}

func (s *MemorySink) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Events)
}

// NewSink returns a sink for the configured audit settings.
func NewSink(cfg config.AuditConfig) (Sink, error) {
	switch cfg.Sink {
	case "stdout", "":
		return WriterSink{w: os.Stdout}, nil
	case "discard":
		return DiscardSink{}, nil
	default:
		return nil, fmt.Errorf("audit.sink %q is not supported", cfg.Sink)
	}
}
