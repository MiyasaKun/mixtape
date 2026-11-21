// Package youtube implements the YouTube source for the music bot. It shells
// out to yt-dlp to extract track metadata and direct audio URLs, and to
// ffmpeg to transcode the audio into raw PCM that is then encoded to Opus
// frames in-process.
package youtube

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"fiber_test/internal/application/ports"

	"github.com/thesyncim/gopus"
)

const (
	sampleRate    = 48000 // Opus frames are always 48kHz
	channels      = 2     // stereo
	frameSize     = 960   // samples per channel per 20ms frame
	pcmFrameWords = frameSize * channels
	pcmFrameBytes = pcmFrameWords * 2 // 20ms of s16le stereo PCM at 48kHz

	// maxYTDLPAttempts is how many times a yt-dlp invocation is retried.
	// YouTube intermittently answers with an empty response / exit 0 when it
	// suspects automation, so retrying makes resolution robust.
	maxYTDLPAttempts = 3
	ytDlpRetryDelay  = 300 * time.Millisecond
)

// Option customises a YouTube adapter.
type Option func(*YouTube)

// WithRunner replaces the yt-dlp execution function. It is intended for
// tests; production always uses the real yt-dlp runner.
func WithRunner(run func(ctx context.Context, args ...string) ([]byte, error)) Option {
	return func(y *YouTube) {
		y.runner = run
	}
}

// WithStreamSpawner replaces the function that starts the yt-dlp → ffmpeg
// process chain. It is intended for tests; production always spawns the
// real chain.
func WithStreamSpawner(spawn func(ctx context.Context, url string) (ports.FrameProvider, error)) Option {
	return func(y *YouTube) {
		y.spawnStream = spawn
	}
}

// WithStreamRetryDelay overrides the pause between stream spawn retries.
// It is intended for tests.
func WithStreamRetryDelay(d time.Duration) Option {
	return func(y *YouTube) {
		y.streamRetryDelay = d
	}
}

// YouTube provides track resolution and audio streaming for YouTube videos
// via the yt-dlp and ffmpeg external binaries.
type YouTube struct {
	runner func(ctx context.Context, args ...string) ([]byte, error)
	// spawnStream starts the audio process chain. The retry loop in Stream
	// calls it up to maxYTDLPAttempts times, because YouTube intermittently
	// rejects the download with 403 (SABR enforcement).
	spawnStream      func(ctx context.Context, url string) (ports.FrameProvider, error)
	streamRetryDelay time.Duration
}

// New returns a YouTube adapter.
func New(opts ...Option) *YouTube {
	y := &YouTube{
		spawnStream:      newYTDLPStreamProvider,
		streamRetryDelay: ytDlpRetryDelay,
	}
	for _, opt := range opts {
		opt(y)
	}
	return y
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
	// --flat-playlist search entries carry the video URL in "url", while
	// full -J dumps use "webpage_url". Either is accepted.
	WebpageURL string `json:"webpage_url"`
	URL        string `json:"url"`
}

// videoURL returns the effective watch URL of the entry, preferring the full
// dump's webpage_url over the flat entry's url field.
func (i ytDlpInfo) videoURL() string {
	if i.WebpageURL != "" {
		return i.WebpageURL
	}
	return i.URL
}

// Resolve extracts metadata for the given video URL (or ytsearch: query).
func (y *YouTube) Resolve(ctx context.Context, url string) (*Track, error) {
	out, err := y.ytdlp(ctx, "--no-playlist", "-J", url)
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

// Search performs a YouTube search via yt-dlp's ytsearch and returns up to
// limit candidate tracks. It uses --flat-playlist so only metadata (no
// formats) is extracted, keeping the call fast.
func (y *YouTube) Search(ctx context.Context, query string, limit int) ([]Track, error) {
	if limit <= 0 {
		limit = 5
	}
	out, err := y.ytdlp(ctx, "--no-playlist", "--flat-playlist", "-J", fmt.Sprintf("ytsearch%d:%s", limit, query))
	if err != nil {
		return nil, errors.New("failed to search youtube: " + err.Error())
	}

	var payload struct {
		Entries []ytDlpInfo `json:"entries"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, errors.New("failed to parse search results: " + err.Error())
	}

	tracks := make([]Track, 0, len(payload.Entries))
	for _, info := range payload.Entries {
		if info.videoURL() == "" {
			continue
		}
		tracks = append(tracks, Track{
			ID:           info.ID,
			Title:        info.Title,
			Artist:       info.Channel,
			Duration:     info.Duration,
			ThumbnailURL: info.Thumbnail,
			URL:          info.videoURL(),
		})
	}
	return tracks, nil
}

// Stream starts yt-dlp downloading the best audio to stdout and pipes it
// into ffmpeg. yt-dlp performs the HTTP fetch itself (headers, PO tokens,
// SABR), because direct ffmpeg fetches of googlevideo URLs are rejected with
// 403 by YouTube.
func (y *YouTube) Stream(ctx context.Context, url string) (ports.FrameProvider, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// The process chain is retried as a whole: YouTube intermittently answers
	// with 403 during extraction/download (SABR enforcement), and a fresh
	// attempt usually succeeds. The spawner only returns a provider once the
	// chain has proven itself (first byte of audio arrived), so a returned
	// error always means no audio was produced and a retry is safe.
	var lastErr error
	for attempt := 0; attempt < maxYTDLPAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(y.streamRetryDelay):
			}
		}
		provider, err := y.spawnStream(ctx, url)
		if err == nil {
			return provider, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// ytdlp runs yt-dlp through the injected runner when one is set (tests),
// otherwise through the real runner.
func (y *YouTube) ytdlp(ctx context.Context, args ...string) ([]byte, error) {
	if y.runner != nil {
		return y.runner(ctx, args...)
	}
	return y.runYTDLP(ctx, args...)
}

// runYTDLP executes yt-dlp with the given arguments and retries on error or
// empty output. Stderr is captured separately so progress messages do not
// corrupt stdout.
func (y *YouTube) runYTDLP(ctx context.Context, args ...string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt < maxYTDLPAttempts; attempt++ {
		var stderr bytes.Buffer
		cmd := exec.CommandContext(ctx, "yt-dlp", args...)
		// The container user cannot write to /, so point yt-dlp's cache at
		// a writable location (see newYTDLPStreamProvider).
		cmd.Env = append(os.Environ(), "XDG_CACHE_HOME=/tmp/yt-dlp-cache", "HOME=/tmp")
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

// streamProvider is a FrameProvider backed by a yt-dlp → ffmpeg process
// chain. yt-dlp downloads the source audio to stdout; ffmpeg transcodes the
// stream to raw s16le PCM at 48kHz stereo; each 20ms chunk is encoded to an
// Opus frame in-process with gopus.
type streamProvider struct {
	ytdlpCmd      *exec.Cmd
	ffmpegCmd     *exec.Cmd
	stdout        io.ReadCloser
	enc           *gopus.Encoder
	processCancel context.CancelFunc
	stderr        *bytes.Buffer
	processErr    error

	done     chan struct{}
	doneOnce sync.Once

	// Reused buffers to avoid allocating per frame.
	pcm      [pcmFrameWords]int16
	pcmBytes [pcmFrameBytes]byte
}

// StreamProbe confirms that audio data is already flowing on r by reading
// its first byte, and returns a reader that replays that byte followed by
// the rest of the stream. Callers must consume the returned reader: using
// r after a successful probe would silently drop the first byte, which
// corrupts the WebM header ffmpeg parses. A probe error means no audio ever
// arrived and the whole process chain should be retried.
func StreamProbe(r io.Reader) (io.Reader, error) {
	buffered := bufio.NewReader(r)
	first, err := buffered.ReadByte()
	if err != nil {
		return nil, err
	}
	return io.MultiReader(bytes.NewReader([]byte{first}), buffered), nil
}

// newYTDLPStreamProvider starts the yt-dlp → ffmpeg process chain and
// returns a provider feeding its PCM output.
//
// It waits for the first byte of audio before returning: if yt-dlp fails
// (for example YouTube's intermittent 403) the error is reported here so
// the retry loop in Stream can start a fresh chain. Once audio has flowed,
// failures are handled by the provider itself.
func newYTDLPStreamProvider(ctx context.Context, url string) (ports.FrameProvider, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// The command that requested playback may finish before the track does.
	// The provider owns both processes after they have been created, so only
	// Close (skip, disconnect, or end of stream) can stop them.
	processCtx, processCancel := context.WithCancel(context.Background())

	// The container user cannot write to /, so point yt-dlp's cache at a
	// writable location. Persisting solved signature functions avoids
	// re-running the JS challenge on every invocation.
	cacheEnv := []string{"XDG_CACHE_HOME=/tmp/yt-dlp-cache", "HOME=/tmp"}

	ytdlpCmd := exec.CommandContext(processCtx, "yt-dlp",
		"--no-playlist", "-f", "bestaudio", "-o", "-", url,
	)
	ytdlpCmd.Env = append(os.Environ(), cacheEnv...)
	ytdlpStdout, err := ytdlpCmd.StdoutPipe()
	if err != nil {
		processCancel()
		return nil, err
	}
	ytdlpStderr := &bytes.Buffer{}
	ytdlpCmd.Stderr = ytdlpStderr

	ffmpegCmd := exec.CommandContext(processCtx, "ffmpeg",
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-i", "pipe:0",
		"-f", "s16le", "-ar", "48000", "-ac", "2",
		"pipe:1",
	)
	ffmpegStdin, err := ffmpegCmd.StdinPipe()
	if err != nil {
		processCancel()
		return nil, err
	}
	ffmpegStdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		processCancel()
		return nil, err
	}
	ffmpegStderr := &bytes.Buffer{}
	ffmpegCmd.Stderr = ffmpegStderr

	if err := ffmpegCmd.Start(); err != nil {
		processCancel()
		return nil, err
	}
	if err := ytdlpCmd.Start(); err != nil {
		processCancel()
		ffmpegCmd.Process.Kill()
		ffmpegCmd.Wait()
		return nil, err
	}

	// Probe phase: wait until yt-dlp either produces its first byte of audio
	// or exits with an error. StreamProbe replays that byte so ffmpeg still
	// receives the complete stream.
	type probeResult struct {
		stream io.Reader
		err    error
	}
	firstByte := make(chan probeResult, 1)
	go func() {
		stream, err := StreamProbe(ytdlpStdout)
		firstByte <- probeResult{stream: stream, err: err}
	}()
	ytdlpDone := make(chan error, 1)
	go func() {
		ytdlpDone <- ytdlpCmd.Wait()
	}()

	var probeErr error
	var audioStream io.Reader
	select {
	case result := <-firstByte:
		probeErr = result.err
		if probeErr == nil {
			audioStream = result.stream // audio is flowing; continue below.
			break
		}
		// Reader error before any audio: treat like an exited process.
		select {
		case waitErr := <-ytdlpDone:
			probeErr = waitErr
		default:
		}
	case waitErr := <-ytdlpDone:
		probeErr = waitErr
	}
	if probeErr != nil {
		processCancel()
		ytdlpCmd.Process.Kill()
		_ = ytdlpCmd.Wait()
		ffmpegCmd.Process.Kill()
		_ = ffmpegCmd.Wait()
		detail := strings.TrimSpace(ytdlpStderr.String())
		if detail == "" {
			detail = probeErr.Error()
		}
		return nil, fmt.Errorf("yt-dlp stream failed: %s", detail)
	}

	// yt-dlp's stdout feeds ffmpeg's stdin. The copy goroutine closes the
	// pipe when the download ends so ffmpeg sees EOF and exits cleanly.
	go func() {
		defer ffmpegStdin.Close()
		_, _ = io.Copy(ffmpegStdin, audioStream)
	}()

	enc, err := gopus.NewEncoder(gopus.EncoderConfig{
		SampleRate:  sampleRate,
		Channels:    channels,
		Application: gopus.ApplicationAudio,
	})
	if err != nil {
		processCancel()
		ytdlpCmd.Process.Kill()
		_ = ytdlpCmd.Wait()
		ffmpegCmd.Process.Kill()
		_ = ffmpegCmd.Wait()
		return nil, err
	}

	return &streamProvider{
		ytdlpCmd:      ytdlpCmd,
		ffmpegCmd:     ffmpegCmd,
		stdout:        ffmpegStdout,
		enc:           enc,
		processCancel: processCancel,
		stderr:        ytdlpStderr,
		done:          make(chan struct{}),
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
		p.finish()
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			if streamErr := p.streamError(); streamErr != nil {
				return nil, streamErr
			}
			return nil, io.EOF
		}
		return nil, err
	}

	// Convert the little-endian s16le bytes into int16 samples.
	for i := 0; i < pcmFrameWords; i++ {
		p.pcm[i] = int16(binary.LittleEndian.Uint16(p.pcmBytes[i*2:]))
	}

	data := make([]byte, 4000)
	n, err := p.enc.EncodeInt16(p.pcm[:], data)
	if err != nil {
		p.finish()
		return nil, err
	}
	return data[:n], nil
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
// and terminates both processes. Killing them also unblocks any read on the
// pipes. ffmpeg's Wait error preserves a real stream failure; a killed
// yt-dlp (whose output ffmpeg already consumed) is not an error.
func (p *streamProvider) finish() {
	p.doneOnce.Do(func() {
		close(p.done)
		p.processCancel()
		if p.ytdlpCmd != nil && p.ytdlpCmd.Process != nil {
			p.ytdlpCmd.Process.Kill()
		}
		_ = p.ytdlpCmd.Wait()
		if p.ffmpegCmd != nil && p.ffmpegCmd.Process != nil {
			p.ffmpegCmd.Process.Kill()
		}
		p.processErr = p.ffmpegCmd.Wait()
	})
}

// streamError preserves ffmpeg's failure instead of making an interrupted
// source look like a normally completed track.
func (p *streamProvider) streamError() error {
	if p.processErr == nil {
		return nil
	}
	if detail := strings.TrimSpace(p.stderr.String()); detail != "" {
		return fmt.Errorf("ffmpeg stream failed: %s", detail)
	}
	return fmt.Errorf("ffmpeg stream failed: %w", p.processErr)
}
