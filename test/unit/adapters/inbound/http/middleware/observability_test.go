package middleware_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"fiber_test/internal/adapters/inbound/http/middleware"
	"fiber_test/internal/infrastructure/logging"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// observed captures one HTTPMetrics.ObserveRequest call.
type observed struct {
	method  string
	route   string
	status  int
	seconds float64
}

type fakeHTTPMetrics struct {
	mu       sync.Mutex
	observes []observed
}

func (f *fakeHTTPMetrics) ObserveRequest(method, route string, status int, seconds float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.observes = append(f.observes, observed{method, route, status, seconds})
}

var _ middleware.HTTPMetrics = (*fakeHTTPMetrics)(nil)

func TestObserveRequestsRecordsMethodRouteStatus(t *testing.T) {
	app := fiber.New()
	metrics := &fakeHTTPMetrics{}
	app.Use(middleware.ObserveRequests(metrics))
	app.Get("/api/tracks/:id", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/tracks/42", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if len(metrics.observes) != 1 {
		t.Fatalf("ObserveRequest calls = %d, want 1", len(metrics.observes))
	}
	o := metrics.observes[0]
	if o.method != http.MethodGet {
		t.Fatalf("method = %q, want GET", o.method)
	}
	if o.route != "/api/tracks/:id" {
		t.Fatalf("route = %q, want route template (not raw URL)", o.route)
	}
	if o.status != http.StatusOK {
		t.Fatalf("status = %d", o.status)
	}
	if o.seconds <= 0 {
		t.Fatalf("seconds = %f, want > 0", o.seconds)
	}
}

func TestObserveRequestsRecordsErrorStatuses(t *testing.T) {
	app := fiber.New()
	metrics := &fakeHTTPMetrics{}
	app.Use(middleware.ObserveRequests(metrics))
	app.Get("/api/gone", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusInternalServerError)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/gone", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if len(metrics.observes) != 1 || metrics.observes[0].status != http.StatusInternalServerError {
		t.Fatalf("observes = %#v", metrics.observes)
	}
}

// requestLogHandler captures log records emitted by the request logger.
type requestLogHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *requestLogHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *requestLogHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *requestLogHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *requestLogHandler) WithGroup(string) slog.Handler { return h }

func TestRequestLoggerEmitsLineWithRequestID(t *testing.T) {
	app := fiber.New()
	handler := &requestLogHandler{}
	appLogger := logging.NewLogger("test", handler)
	app.Use(requestid.New())
	app.Use(middleware.RequestLog(appLogger))
	app.Get("/api/health", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()
	if len(handler.records) != 1 {
		t.Fatalf("log records = %d, want 1", len(handler.records))
	}
	record := handler.records[0]
	if record.Message != "http request" {
		t.Fatalf("message = %q, want %q", record.Message, "http request")
	}

	m := make(map[string]string)
	record.Attrs(func(a slog.Attr) bool {
		if a.Value.Kind() == slog.KindString {
			m[a.Key] = a.Value.String()
		}
		return true
	})
	if got := m["request_id"]; got == "" {
		t.Fatalf("request_id attribute missing, attrs = %v", m)
	}
	if got := m["method"]; got != http.MethodGet {
		t.Fatalf("method attr = %q", got)
	}
}

func TestRequestLoggerEscalatesServerErrorToError(t *testing.T) {
	app := fiber.New()
	handler := &requestLogHandler{}
	appLogger := logging.NewLogger("test", handler)
	app.Use(requestid.New())
	app.Use(middleware.RequestLog(appLogger))
	app.Get("/api/boom", func(c fiber.Ctx) error {
		time.Sleep(0)
		return c.SendStatus(http.StatusInternalServerError)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/boom", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()
	if len(handler.records) != 1 {
		t.Fatalf("log records = %d, want 1", len(handler.records))
	}
	if lvl := handler.records[0].Level; lvl != slog.LevelError {
		t.Fatalf("level = %v, want error for 5xx", lvl)
	}
}
