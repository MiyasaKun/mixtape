package ports

import "context"

// AudioConn is the destination for audio playback. The application layer
// hands a FrameProvider to the connection; the concrete implementation is
// responsible for wiring it into the Discord voice pipeline.
type AudioConn interface {
	// SetFrameProvider starts playback of the given provider, replacing any
	// currently playing provider.
	SetFrameProvider(provider FrameProvider)

	// Close disconnects the bot from the voice channel.
	Close(ctx context.Context)
}
