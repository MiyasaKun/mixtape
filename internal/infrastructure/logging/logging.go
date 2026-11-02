package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

type Logger struct {
	*slog.Logger
}

var Log *slog.Logger

// NewLogger builds the application logger. The console handler is chosen by
// env (JSON in production, colored text otherwise). Additional handlers are
// fanned out as secondary sinks alongside the console handler, which lets
// observability bridges (e.g. OpenTelemetry) see every record without leaving
// the console output unchanged.
func NewLogger(env string, extraHandlers ...slog.Handler) *Logger {
	var console slog.Handler

	if env == "production" {
		console = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		console = tint.NewTextHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.RFC1123,
		})
	}

	handler := fanout(console, extraHandlers...)

	Log = slog.New(handler)

	return &Logger{
		Logger: slog.New(handler),
	}
}

// NewNoopLogger returns a logger that discards every record. Use it to keep
// services that take an optional logger safe to construct without one (e.g.
// in tests).
func NewNoopLogger() *Logger {
	return &Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Error(msg, args...)
	os.Exit(1)
}

// fanoutHandler dispatches records to the console handler and any number of
// extra sinks. It is enabled if any of the underlying handlers is.
type fanoutHandler struct {
	handlers []slog.Handler
}

func fanout(console slog.Handler, extra ...slog.Handler) slog.Handler {
	if len(extra) == 0 {
		return console
	}
	handlers := make([]slog.Handler, 0, len(extra)+1)
	handlers = append(handlers, console)
	handlers = append(handlers, extra...)
	return fanoutHandler{handlers: handlers}
}

func (f fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (f fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range f.handlers {
		if err := h.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (f fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return fanoutHandler{handlers: handlers}
}

func (f fanoutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return fanoutHandler{handlers: handlers}
}
