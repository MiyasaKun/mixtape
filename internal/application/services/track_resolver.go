package services

import (
	"context"
	"errors"
	"strings"

	"fiber_test/internal/adapters/outbound/soundcloud"
	"fiber_test/internal/adapters/outbound/spotify"
	"fiber_test/internal/adapters/outbound/youtube"
)

// TrackResolverService turns a user-supplied URL or search query into a
// Track by routing it to the adapter matching the source platform.
type TrackResolverService struct {
	youtube    *youtube.YouTube
	spotify    *spotify.Spotify
	soundcloud *soundcloud.SoundCloud
}

// NewTrackResolverService builds a resolver from the per-platform adapters.
func NewTrackResolverService(yt *youtube.YouTube, sp *spotify.Spotify, sc *soundcloud.SoundCloud) *TrackResolverService {
	return &TrackResolverService{
		youtube:    yt,
		spotify:    sp,
		soundcloud: sc,
	}
}

// Resolve detects the source of the query and resolves it to a Track.
// Recognized sources are YouTube (URL or plain search query, resolved via
// yt-dlp's ytsearch). Spotify and SoundCloud are not yet supported for
// playback.
func (s *TrackResolverService) Resolve(ctx context.Context, query string) (*Track, error) {
	switch {
	case isSpotifyURL(query):
		return nil, errors.New("spotify tracks are not yet supported for playback")
	case isSoundCloudURL(query):
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
	ytTrack, err := s.youtube.Resolve(ctx, url)
	if err != nil {
		return nil, err
	}

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
