// Package ports defines the boundaries between the application layer and the
// outside world. The interfaces here are implemented by outbound adapters
// (e.g. the YouTube adapter) and consumed by inbound adapters (e.g. the
// Discord command layer).
package ports

import "context"

// FrameProvider is a stream of Opus audio frames destined for a voice
// connection. The Discord audio sender pulls frames via ProvideFrame at a
// fixed 20ms cadence.
type FrameProvider interface {
	// ProvideFrame returns the next Opus frame, or io.EOF once the stream
	// has finished. Implementations must keep returning io.EOF (not errors)
	// after the stream ends.
	ProvideFrame() ([]byte, error)

	// Done is closed exactly once when the stream ends, whether naturally
	// or because Close was called. It lets the playback service react to a
	// finished track and advance the queue.
	Done() <-chan struct{}

	// Close stops the underlying stream (e.g. kills the ffmpeg process) and
	// releases resources. It is idempotent and also fires Done.
	Close()
}

// AudioStreamer turns a playable URL into a FrameProvider.
type AudioStreamer interface {
	// Stream resolves the URL and returns a provider producing Opus frames
	// from the resolved audio.
	Stream(ctx context.Context, url string) (FrameProvider, error)
}
