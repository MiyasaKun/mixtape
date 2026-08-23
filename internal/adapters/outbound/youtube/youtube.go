// Package youtube implements the YouTube source for the music bot. It shells
// out to yt-dlp to extract track metadata and direct audio URLs, and to
// ffmpeg to transcode the audio into raw PCM that is then encoded to Opus
// frames in-process.
package youtube

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"fiber_test/internal/application/ports"

	"layeh.com/gopus"
)

const (
	sampleRate    = 48000 // Opus frames are always 48kHz
	channels      = 2     // stereo
	frameSize     = 960   // samples per channel per 20ms frame
	pcmFrameBytes = 3840  // 20ms of s16le stereo PCM at 48kHz
	pcmFrameWords = 1920  // 20ms of int16 stereo PCM at 48kHz

	// maxYTDLPAttempts is how many times a yt-dlp invocation is retried.
	// YouTube intermittently answers with an empty response / exit 0 when it
	// suspects automation, so retrying makes resolution robust.
	maxYTDLPAttempts = 3
	ytDlpRetryDelay  = 300 * time.Millisecond
)

// YouTube provides track resolution and audio streaming for YouTube videos
// via the yt-dlp and ffmpeg external binaries.
type YouTube struct {
}

// New returns a YouTube adapter.
func New() *YouTube {
	return &YouTube{}
}

// Track holds the metadata of a resolved YouTube video.
type Track struct {
	ID           string
	Title        string
	Artist       string
	Duration     int64
	ThumbnailURL string
	URL          string
}

// ytDlpInfo maps the subset of yt-dlp's JSON output that we care about.
// yt-dlp is invoked with -J which dumps the full info JSON to stdout.
type ytDlpInfo struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Channel    string `json:"channel"`
	Duration   int64  `json:"duration"`
	Thumbnail  string `json:"thumbnail"`
	WebpageURL string `json:"webpage_url"`
}

// Resolve extracts metadata for the given video URL (or ytsearch: query).
func (y *YouTube) Resolve(ctx context.Context, url string) (*Track, error) {
	out, err := y.runYTDLP(ctx, "--no-playlist", "-J", url)
	if err != nil {
		return nil, errors.New("failed to resolve track metadata: " + err.Error())
	}

	var info ytDlpInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, errors.New("failed to parse track metadata: " + err.Error())
	}

	return &Track{
		ID:           info.ID,
		Title:        info.Title,
		Artist:       info.Channel,
		Duration:     info.Duration,
		ThumbnailURL: info.Thumbnail,
		URL:          info.WebpageURL,
	}, nil
}

// Stream resolves the direct audio URL of the given video and returns an
// Opus FrameProvider for it.
func (y *YouTube) Stream(ctx context.Context, url string) (ports.FrameProvider, error) {
	streamURL, err := y.resolveStreamURL(ctx, url)
	if err != nil {
		return nil, err
	}
	return newStreamProvider(ctx, streamURL)
}

// runYTDLP executes yt-dlp with the given arguments and retries on error or
// empty output. Stderr is captured separately so progress messages do not
// corrupt stdout.
func (y *YouTube) runYTDLP(ctx context.Context, args ...string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt < maxYTDLPAttempts; attempt++ {
		var stderr bytes.Buffer
		cmd := exec.CommandContext(ctx, "yt-dlp", args...)
		cmd.Stderr = &stderr

		out, err := cmd.Output()
		switch {
		case err != nil:
			lastErr = errors.New(strings.TrimSpace(stderr.String()))
		case len(bytes.TrimSpace(out)) == 0:
			lastErr = errors.New("yt-dlp returned no output")
		default:
			return out, nil
		}

		// Bail out immediately if the context is gone.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		select {
		case <-time.After(ytDlpRetryDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, lastErr
}

// resolveStreamURL runs yt-dlp -g to print the direct URL of the best audio
// format. The first non-empty output line is used so stray extra lines (e.g.
// from playlist URLs) cannot corrupt the result.
func (y *YouTube) resolveStreamURL(ctx context.Context, url string) (string, error) {
	out, err := y.runYTDLP(ctx, "--no-playlist", "-f", "bestaudio", "-g", url)
	if err != nil {
		return "", errors.New("failed to resolve audio stream: " + err.Error())
	}

	streamURL := firstNonEmptyLine(out)
	if streamURL == "" {
		return "", errors.New("yt-dlp returned no audio stream URL")
	}
	return streamURL, nil
}

// firstNonEmptyLine returns the first non-whitespace line of b.
func firstNonEmptyLine(b []byte) string {
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// streamProvider is a FrameProvider backed by an ffmpeg process. ffmpeg
// transcodes the source audio to raw s16le PCM at 48kHz stereo on stdout;
// each 20ms chunk is encoded to an Opus frame in-process with gopus.
type streamProvider struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	enc    *gopus.Encoder

	done     chan struct{}
	doneOnce sync.Once

	// Reused buffers to avoid allocating per frame.
	pcm      [pcmFrameWords]int16
	pcmBytes [pcmFrameBytes]byte
}

// newStreamProvider starts ffmpeg decoding the given stream URL and returns
// a provider feeding its PCM output.
func newStreamProvider(ctx context.Context, streamURL string) (ports.FrameProvider, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-i", streamURL,
		"-f", "s16le", "-ar", "48000", "-ac", "2",
		"pipe:1",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	enc, err := gopus.NewEncoder(sampleRate, channels, gopus.Audio)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, err
	}

	return &streamProvider{
		cmd:    cmd,
		stdout: stdout,
		enc:    enc,
		done:   make(chan struct{}),
	}, nil
}

// ProvideFrame reads the next 20ms of PCM, encodes it to Opus and returns the
// frame. Once the stream is finished (or closed) it always returns io.EOF.
func (p *streamProvider) ProvideFrame() ([]byte, error) {
	// Return EOF immediately after the provider has finished or been closed.
	select {
	case <-p.done:
		return nil, io.EOF
	default:
	}

	// ReadFull ensures we only encode a complete 20ms frame; anything short
	// means the stream ended mid-frame.
	if _, err := io.ReadFull(p.stdout, p.pcmBytes[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			p.finish()
			return nil, io.EOF
		}
		return nil, err
	}

	// Convert the little-endian s16le bytes into int16 samples.
	for i := 0; i < pcmFrameWords; i++ {
		p.pcm[i] = int16(binary.LittleEndian.Uint16(p.pcmBytes[i*2:]))
	}

	return p.enc.Encode(p.pcm[:], frameSize, 4000)
}

// Done reports when the stream has finished or been closed.
func (p *streamProvider) Done() <-chan struct{} {
	return p.done
}

// Close stops the stream and kills the underlying ffmpeg process.
func (p *streamProvider) Close() {
	p.finish()
}

// finish tears down the provider exactly once: it closes the done channel
// and terminates ffmpeg. Killing ffmpeg also unblocks any read on stdout.
func (p *streamProvider) finish() {
	p.doneOnce.Do(func() {
		close(p.done)
		if p.cmd != nil && p.cmd.Process != nil {
			p.cmd.Process.Kill()
			p.cmd.Wait()
		}
	})
}
