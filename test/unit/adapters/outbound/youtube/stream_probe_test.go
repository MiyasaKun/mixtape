package youtube_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"fiber_test/internal/adapters/outbound/youtube"
)

// TestStreamProbePreservesAllBytes guards the pipe into ffmpeg: the probe
// consumes yt-dlp's first byte (the WebM magic 0x1A) to confirm audio is
// flowing, and the returned reader must replay it. Dropping it makes ffmpeg
// see a corrupt stream ("Invalid data found when processing input").
func TestStreamProbePreservesAllBytes(t *testing.T) {
	payload := "\x1aE\xdf\xa3" + strings.Repeat("x", 8000) // WebM header prefix
	reader, err := youtube.StreamProbe(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("StreamProbe returned error: %v", err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read probed stream: %v", err)
	}
	if string(got) != payload {
		t.Errorf("probed stream lost %d bytes (want %d, got %d); first byte = 0x%02x, want 0x1a",
			len(payload)-len(got), len(payload), len(got), got[0])
	}
}

// TestStreamProbeReportsEmptyStream verifies a stream that produces no audio
// fails fast so the retry loop in Stream can start a fresh process chain.
func TestStreamProbeReportsEmptyStream(t *testing.T) {
	reader, err := youtube.StreamProbe(strings.NewReader(""))
	if err == nil {
		t.Fatal("want error for a stream with no data")
	}
	if reader != nil {
		t.Error("want nil reader on probe failure")
	}
	if !errors.Is(err, io.EOF) {
		t.Errorf("err = %v, want io.EOF", err)
	}
}
