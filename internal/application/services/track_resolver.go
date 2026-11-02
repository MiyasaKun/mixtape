package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"fiber_test/internal/application/ports"
	"fiber_test/internal/infrastructure/logging"

	"fiber_test/internal/adapters/outbound/soundcloud"
	"fiber_test/internal/adapters/outbound/spotify"
	"fiber_test/internal/adapters/outbound/youtube"
)

// YouTubeResolver resolves YouTube URLs and search queries to adapter
// tracks. The interface keeps the resolver testable without yt-dlp.
type YouTubeResolver interface {
	Resolve(ctx context.Context, url string) (*youtube.Track, error)
}

// TrackResolverService turns a user-supplied URL or search query into a
// Track by routing it to the adapter matching the source platform.
type TrackResolverService struct {
	youtube    YouTubeResolver
	spotify    *spotify.Spotify
	soundcloud *soundcloud.SoundCloud
	logger     *logging.Logger
	metrics    ports.Metrics
}

// NewTrackResolverService builds a resolver from the per-platform adapters.
// Telemetry is an optional trailing option; without it the service logs to a
// discard sink and emits no metrics.
func NewTrackResolverService(yt YouTubeResolver, sp *spotify.Spotify, sc *soundcloud.SoundCloud, telem ...Telemetry) *TrackResolverService {
	deps := resolveTelemetry(telem)
	return &TrackResolverService{
		youtube:    yt,
		spotify:    sp,
		soundcloud: sc,
		logger:     deps.logger,
		metrics:    deps.metrics,
	}
}

// Resolve detects the source of the query and resolves it to a Track.
// Recognized sources are YouTube (URL or plain search query, resolved via
// yt-dlp's ytsearch). Spotify and SoundCloud are not yet supported for
// playback.
func (s *TrackResolverService) Resolve(ctx context.Context, query string) (*Track, error) {
	switch {
	case isSpotifyURL(query):
		s.metrics.TrackResolved("spotify", "unsupported")
		s.logger.Warn("unsupported source requested", "provider", "spotify", "query", query)
		return nil, errors.New("spotify tracks are not yet supported for playback")
	case isSoundCloudURL(query):
		s.metrics.TrackResolved("soundcloud", "unsupported")
		s.logger.Warn("unsupported source requested", "provider", "soundcloud", "query", query)
		return nil, errors.New("soundcloud tracks are not yet supported for playback")
	case isYouTubeURL(query):
		return s.resolveYouTube(ctx, query)
	default:
		// Treat plain text as a YouTube search; yt-dlp picks the first hit.
		return s.resolveYouTube(ctx, "ytsearch:"+query)
	}
}

// resolveYouTube fetches metadata from the YouTube adapter and maps it onto
// the application-level Track type.
func (s *TrackResolverService) resolveYouTube(ctx context.Context, url string) (*Track, error) {
	start := time.Now()
	ytTrack, err := s.youtube.Resolve(ctx, url)
	s.metrics.TrackResolutionDuration("youtube", time.Since(start).Seconds())
	if err != nil {
		s.metrics.TrackResolved("youtube", "error")
		s.logger.Error("youtube track resolution failed", "url", url, "error", err)
		return nil, fmt.Errorf("resolve youtube track %q: %w", url, err)
	}
	s.metrics.TrackResolved("youtube", "ok")
	s.logger.Debug("youtube track resolved", "url", url, "title", ytTrack.Title)

	return &Track{
		ID:           ytTrack.ID,
		Title:        ytTrack.Title,
		Artist:       ytTrack.Artist,
		Duration:     ytTrack.Duration,
		Source:       "youtube",
		URL:          ytTrack.URL,
		ThumbnailURL: ytTrack.ThumbnailURL,
	}, nil
}

func isYouTubeURL(query string) bool {
	lower := strings.ToLower(query)
	return strings.Contains(lower, "youtube.com/") || strings.Contains(lower, "youtu.be/")
}

func isSpotifyURL(query string) bool {
	return strings.HasPrefix(strings.ToLower(query), "https://open.spotify.com/")
}

func isSoundCloudURL(query string) bool {
	lower := strings.ToLower(query)
	return strings.Contains(lower, "soundcloud.com/")
}
