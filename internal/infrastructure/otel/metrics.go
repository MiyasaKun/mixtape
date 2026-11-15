package otel

import (
	"context"
	"errors"

	"fiber_test/internal/application/ports"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// recorder implements ports.Metrics on top of an OpenTelemetry meter.
// Instruments are created eagerly; recording is a single add on the hot path.
type recorder struct {
	commands        metric.Int64Counter
	trackResolved   metric.Int64Counter
	resolution      metric.Float64Histogram
	playbackFailure metric.Int64Counter
	depth           metric.Int64Gauge
}

const scopeName = "fiber_test/mixtape"

// newMetricsRecorder creates the instrument set. It fails only on invalid
// instrument names (static strings here), so callers can treat errors as a
// "telemetry unavailable" case and fall back to a noop.
func newMetricsRecorder(mp *sdkmetric.MeterProvider) (ports.Metrics, error) {
	meter := mp.Meter(scopeName)

	commands, err := meter.Int64Counter("discord_command_total",
		metric.WithDescription("Slash commands executed, by command and outcome."))
	if err != nil {
		return nil, errors.Join(errors.New("otel: creating discord_command_total"), err)
	}

	tracksResolved, err := meter.Int64Counter("track_resolved_total",
		metric.WithDescription("Track resolution attempts, by provider and outcome."))
	if err != nil {
		return nil, errors.Join(errors.New("otel: creating track_resolved_total"), err)
	}

	resolution, err := meter.Float64Histogram("track_resolution_seconds",
		metric.WithUnit("s"),
		metric.WithDescription("Time a provider took to resolve a track."))
	if err != nil {
		return nil, errors.Join(errors.New("otel: creating track_resolution_seconds"), err)
	}

	playbackFailures, err := meter.Int64Counter("playback_failures_total",
		metric.WithDescription("Failed playback attempts, by pipeline stage."))
	if err != nil {
		return nil, errors.Join(errors.New("otel: creating playback_failures_total"), err)
	}

	depth, err := meter.Int64Gauge("queue_depth",
		metric.WithDescription("Number of queued tracks, by guild."))
	if err != nil {
		return nil, errors.Join(errors.New("otel: creating queue_depth"), err)
	}

	return &recorder{
		commands:        commands,
		trackResolved:   tracksResolved,
		resolution:      resolution,
		playbackFailure: playbackFailures,
		depth:           depth,
	}, nil
}

func attrs(kv ...string) metric.MeasurementOption {
	o := make([]attribute.KeyValue, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		o = append(o, attribute.String(kv[i], kv[i+1]))
	}
	return metric.WithAttributes(o...)
}

func (r *recorder) CommandHandled(command, status string) {
	r.commands.Add(context.Background(), 1, attrs("command", command, "status", status))
}

func (r *recorder) TrackResolved(provider, status string) {
	r.trackResolved.Add(context.Background(), 1, attrs("provider", provider, "status", status))
}

func (r *recorder) TrackResolutionDuration(provider string, seconds float64) {
	r.resolution.Record(context.Background(), seconds, attrs("provider", provider))
}

func (r *recorder) PlaybackFailure(reason string) {
	r.playbackFailure.Add(context.Background(), 1, attrs("reason", reason))
}

func (r *recorder) QueueDepth(guildID string, depth int) {
	r.depth.Record(context.Background(), int64(depth), attrs("guild", guildID))
}

var _ ports.Metrics = (*recorder)(nil)
