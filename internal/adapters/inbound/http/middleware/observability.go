package middleware

import (
	"log/slog"
	"time"

	"fiber_test/internal/infrastructure/logging"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// HTTPMetrics records observed HTTP traffic. It is defined here (a pure
// inbound-middleware concern) so implementations can live in the
// infrastructure layer without the middleware importing it.
type HTTPMetrics interface {
	ObserveRequest(method, route string, status int, seconds float64)
}

// ObserveRequests measures every request's duration and hands it to the
// metrics recorder together with the matched *route template* (e.g.
// /api/tracks/:id, never a raw URL — raw values would blow up label
// cardinality). Chain it with requestid so correlation data exists.
func ObserveRequests(metrics HTTPMetrics) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		// c.Route() is nil for requests that fell through to the SPA static
		// handler; fall back to the raw path in that case.
		route := c.Path()
		if r := c.Route(); r != nil && r.Path != "" {
			route = r.Path
		}
		metrics.ObserveRequest(c.Method(), route, c.Response().StatusCode(), time.Since(start).Seconds())

		return err
	}
}

// RequestLog writes one structured log line per request, escalating warning
// and error levels for slow- or server-side failures. Browser-visible client
// errors stay at debug level so dashboards and stdout stay readable.
func RequestLog(appLogger *logging.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		route := c.Path()
		if r := c.Route(); r != nil && r.Path != "" {
			route = r.Path
		}

		ctx := c.RequestCtx()
		status := c.Response().StatusCode()
		level := slog.LevelDebug
		switch {
		case status >= httpStatusServerError:
			level = slog.LevelError
		case status >= httpStatusClientError:
			level = slog.LevelWarn
		}

		appLogger.Log(ctx, level, "http request",
			"request_id", requestid.FromContext(c),
			"method", c.Method(),
			"path", c.Path(),
			"route", route,
			"status", status,
			"latency_ms", time.Since(start).Milliseconds())

		return err
	}
}

const (
	httpStatusClientError = 400
	httpStatusServerError = 500
)
