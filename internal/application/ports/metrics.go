package ports

// Metrics is the telemetry boundary for application services. Infrastructure
// adapters implement it on top of OpenTelemetry; tests pass fakes or a
// NoopMetrics. Callers never need nil checks: a disabled telemetry setup
// hands out a NoopMetrics instead of a nil interface.
type Metrics interface {
	// CommandHandled counts a slash command outcome. status is "ok", "error"
	// or "unknown".
	CommandHandled(command, status string)

	// TrackResolved counts a resolver outcome for a source provider.
	TrackResolved(provider, status string)

	// TrackResolutionDuration records how long a provider took to resolve a
	// track.
	TrackResolutionDuration(provider string, seconds float64)

	// PlaybackFailure counts a failed playback attempt, tagged with the
	// pipeline stage that failed.
	PlaybackFailure(reason string)

	// QueueDepth reports the current queue size for a guild.
	QueueDepth(guildID string, depth int)
}

// NoopMetrics discards every telemetry event. Use it when telemetry is
// disabled so code can always assume a non-nil Metrics.
type NoopMetrics struct{}

// NewNoopMetrics returns a Metrics that drops everything.
func NewNoopMetrics() Metrics {
	return NoopMetrics{}
}

// CommandHandled implements Metrics. It is a no-op.
func (NoopMetrics) CommandHandled(string, string) {}

// TrackResolved implements Metrics. It is a no-op.
func (NoopMetrics) TrackResolved(string, string) {}

// TrackResolutionDuration implements Metrics. It is a no-op.
func (NoopMetrics) TrackResolutionDuration(string, float64) {}

// PlaybackFailure implements Metrics. It is a no-op.
func (NoopMetrics) PlaybackFailure(string) {}

// QueueDepth implements Metrics. It is a no-op.
func (NoopMetrics) QueueDepth(string, int) {}
