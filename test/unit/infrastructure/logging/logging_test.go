package logging_test

import (
	"context"
	"log/slog"
	"testing"

	"fiber_test/internal/infrastructure/logging"
)

// fanoutHandler is a slog.Handler that can hold the last record it received.
type recordingHandler struct {
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *recordingHandler) WithGroup(string) slog.Handler {
	return h
}

func TestNewLoggerWritesToExtraHandlers(t *testing.T) {
	extra := &recordingHandler{}
	logger := logging.NewLogger("development", extra)

	logger.With("component", "test").Info("hello", "key", "value")

	if len(extra.records) != 1 {
		t.Fatalf("extra handler received %d records, want 1", len(extra.records))
	}
	_record := extra.records[0]
	if _record.Message != "hello" {
		t.Fatalf("record message = %q, want %q", _record.Message, "hello")
	}
	if !extra.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("extra handler should stay enabled")
	}
}
