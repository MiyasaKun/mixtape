package youtube_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"fiber_test/internal/adapters/outbound/youtube"

	"fiber_test/internal/application/ports"
)

// fakeFrameProvider is a stand-in for the yt-dlp → ffmpeg provider so tests
// can observe spawn attempts without launching real processes.
type fakeFrameProvider struct{}

func (fakeFrameProvider) ProvideFrame() ([]byte, error) { return nil, nil }
func (fakeFrameProvider) Done() <-chan struct{}         { return nil }
func (fakeFrameProvider) Close()                        {}

// TestStreamRetriesWhenSpawnFails verifies that a transient yt-dlp failure
// (e.g. YouTube's intermittent 403 during extraction) retries the whole
// process chain instead of killing playback right away.
func TestStreamRetriesWhenSpawnFails(t *testing.T) {
	attempts := 0
	client := youtube.New(
		youtube.WithStreamSpawner(func(ctx context.Context, url string) (ports.FrameProvider, error) {
			attempts++
			if attempts < 3 {
				return nil, errors.New("yt-dlp stream failed: HTTP Error 403: Forbidden")
			}
			return fakeFrameProvider{}, nil
		}),
		youtube.WithStreamRetryDelay(0),
	)

	provider, err := client.Stream(context.Background(), "https://www.youtube.com/watch?v=x")
	if err != nil {
		t.Fatalf("Stream returned error after retries: %v", err)
	}
	if provider == nil {
		t.Fatal("Stream returned nil provider")
	}
	if attempts != 3 {
		t.Errorf("spawn attempts = %d, want 3", attempts)
	}
}

// TestStreamFailsAfterMaxAttempts verifies the retry loop gives up after the
// configured number of attempts and reports the last error.
func TestStreamFailsAfterMaxAttempts(t *testing.T) {
	wantErr := errors.New("yt-dlp stream failed: HTTP Error 403: Forbidden")
	attempts := 0
	client := youtube.New(
		youtube.WithStreamSpawner(func(ctx context.Context, url string) (ports.FrameProvider, error) {
			attempts++
			return nil, wantErr
		}),
		youtube.WithStreamRetryDelay(0),
	)

	_, err := client.Stream(context.Background(), "https://www.youtube.com/watch?v=x")
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if attempts != 3 {
		t.Errorf("spawn attempts = %d, want 3", attempts)
	}
}

// TestStreamRespectsContextCancellation verifies the retry loop stops
// immediately when the caller's context is cancelled.
func TestStreamRespectsContextCancellation(t *testing.T) {
	attempts := 0
	client := youtube.New(
		youtube.WithStreamSpawner(func(ctx context.Context, url string) (ports.FrameProvider, error) {
			attempts++
			return nil, errors.New("boom")
		}),
		youtube.WithStreamRetryDelay(50*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()
	if _, err := client.Stream(ctx, "https://www.youtube.com/watch?v=x"); err == nil {
		t.Fatal("want error when context is cancelled")
	}
	if attempts > 3 {
		t.Errorf("spawn attempts = %d, want at most 3", attempts)
	}
}
