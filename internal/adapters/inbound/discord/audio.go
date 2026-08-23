package discord

import (
	"fiber_test/internal/application/ports"

	"github.com/disgoorg/disgo/voice"
)

// voiceConnAdapter adapts a disgo voice.Conn to the application-level
// ports.AudioConn interface. It is the bridge between the playback service
// and the Discord voice pipeline.
type voiceConnAdapter struct {
	conn voice.Conn
}

// NewVoiceConnAdapter wraps a disgo voice.Conn so it can be used as a
// ports.AudioConn by the music playback service.
func NewVoiceConnAdapter(conn voice.Conn) ports.AudioConn {
	return &voiceConnAdapter{conn: conn}
}

func (a *voiceConnAdapter) SetFrameProvider(provider ports.FrameProvider) {
	a.conn.SetOpusFrameProvider(&opusFrameProvider{provider: provider})
}

// opusFrameProvider adapts a ports.FrameProvider to the OpusFrameProvider
// interface disgo's audio sender expects.
type opusFrameProvider struct {
	provider ports.FrameProvider
}

func (p *opusFrameProvider) ProvideOpusFrame() ([]byte, error) {
	return p.provider.ProvideFrame()
}

func (p *opusFrameProvider) Close() {
	p.provider.Close()
}
