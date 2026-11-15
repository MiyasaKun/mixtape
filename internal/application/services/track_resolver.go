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

// spotifyChoiceCount is how many YouTube candidates a Spotify link offers
// for the user to pick from.
const spotifyChoiceCount = 5

// YouTubeResolver resolves YouTube URLs and search queries to adapter
// tracks. The interface keeps the resolver testable without yt-dlp.
type YouTubeResolver interface {
	Resolve(ctx context.Context, url string) (*youtube.Track, error)
}

// YouTubeSearcher runs YouTube queries for candidate tracks (e.g. to map a
// Spotify link onto playable YouTube results). Optional capability: resolvers
// backed by a search-less adapter simply leave it unimplemented.
type YouTubeSearcher interface {
	Search(ctx context.Context, query string, limit int) ([]youtube.Track, error)
}

// TrackResolverService turns a user-supplied URL or search query into a
// Track by routing it to the adapter matching the source platform.
type TrackResolverService struct {
	youtube       YouTubeResolver
	youtubeSearch YouTubeSearcher
	spotify       *spotify.Spotify
	soundcloud    *soundcloud.SoundCloud
	logger        *logging.Logger
	metrics       ports.Metrics
}

// NewTrackResolverService builds a resolver from the per-platform adapters.
// Telemetry is an optional trailing option; without it the service logs to a
// discard sink and emits no metrics.
func NewTrackResolverService(yt YouTubeResolver, sp *spotify.Spotify, sc *soundcloud.SoundCloud, telem ...Telemetry) *TrackResolverService {
	deps := resolveTelemetry(telem)
	resolver := &TrackResolverService{
		youtube:    yt,
		spotify:    sp,
		soundcloud: sc,
		logger:     deps.logger,
		metrics:    deps.metrics,
	}
	// The search capability is derived from the same adapter: the real
	// YouTube client implements both interfaces, fakes in tests only Resolve.
	if searcher, ok := yt.(YouTubeSearcher); ok {
		resolver.youtubeSearch = searcher
	}
	return resolver
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

// IsSpotifyURL reports whether the query is a Spotify open.spotify.com URL.
// The scheme is optional — Discord sometimes strips it from pasted links.
func IsSpotifyURL(query string) bool {
	host := strings.ToLower(strings.TrimSpace(query))
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "www.")
	return strings.HasPrefix(host, "open.spotify.com/")
}

// ExtractSpotifyTrackID pulls the track ID out of an
// https://open.spotify.com/track/<id> URL (scheme optional), tolerating the
// optional /intl-<locale>/ path segment Spotify inserts for localized links.
// It returns "" for other Spotify URLs (albums, playlists, …).
func ExtractSpotifyTrackID(rawURL string) string {
	// Spotify track IDs are base62 and case-sensitive — never lowercase the
	// URL, only normalize the path segments we compare against.
	u := strings.TrimSpace(rawURL)
	if !IsSpotifyURL(u) {
		return ""
	}
	// Strip scheme + www (case-insensitively) but keep the rest verbatim.
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(strings.ToLower(u), prefix) {
			u = u[len(prefix):]
			break
		}
	}
	u = strings.TrimPrefix(u, "www.")
	u = u[len("open.spotify.com/"):]

	// Skip the optional intl-<locale> path segment(s).
	for hasIntlPrefix(u) {
		u = u[len("intl-"):]
		if i := strings.Index(u, "/"); i != -1 {
			u = u[i+1:]
		} else {
			return ""
		}
	}
	if !strings.HasPrefix(strings.ToLower(u), "track/") {
		return ""
	}
	id := strings.TrimPrefix(u, "track/")
	if i := strings.IndexAny(id, "?#"); i != -1 {
		id = id[:i]
	}
	if id == "" || strings.Contains(id, "/") {
		return ""
	}
	return id
}

// SpotifyYouTubeOptions resolves a Spotify track link into candidate YouTube
// tracks for the user to pick from: the track's metadata is looked up via the
// Spotify Web API (app credentials), and an "artists - title" search is run
// against YouTube via yt-dlp, returning up to 5 candidates. It also returns
// the track's Spotify album image so the /nowplaying panel can show it.
func (s *TrackResolverService) SpotifyYouTubeOptions(ctx context.Context, spotifyURL string) ([]Track, string, error) {
	trackID := ExtractSpotifyTrackID(spotifyURL)
	if trackID == "" {
		return nil, "", errors.New("only Spotify track links are supported, e.g. https://open.spotify.com/track/<id>")
	}
	if s.spotify == nil || s.youtubeSearch == nil {
		return nil, "", errors.New("spotify or youtube support is not configured")
	}

	spTrack, err := s.spotify.GetTrack(ctx, trackID)
	if err != nil {
		s.logger.Warn("spotify track lookup failed", "spotify_url", spotifyURL, "error", err)
		return nil, "", fmt.Errorf("failed to fetch spotify track: %w", err)
	}

	query := strings.Join(spTrack.Artists, ", ")
	if query != "" {
		query += " - "
	}
	query += spTrack.Title

	candidates, err := s.youtubeSearch.Search(ctx, query, spotifyChoiceCount)
	if err != nil {
		s.logger.Warn("youtube search for spotify candidates failed", "query", query, "error", err)
		return nil, "", fmt.Errorf("failed to search youtube: %w", err)
	}

	tracks := make([]Track, 0, len(candidates))
	for _, yt := range candidates {
		tracks = append(tracks, mapYouTubeTrack(yt))
	}
	return tracks, spTrack.ImageURL, nil
}

func mapYouTubeTrack(yt youtube.Track) Track {
	return Track{
		ID:           yt.ID,
		Title:        yt.Title,
		Artist:       yt.Artist,
		Duration:     yt.Duration,
		Source:       "youtube",
		URL:          yt.URL,
		ThumbnailURL: yt.ThumbnailURL,
	}
}

// hasIntlPrefix reports whether the remaining path starts with the intl-
// locale segment (compared case-insensitively, like the scheme prefix).
func hasIntlPrefix(u string) bool {
	if len(u) < len("intl-") {
		return false
	}
	return strings.EqualFold(u[:len("intl-")], "intl-")
}

func isYouTubeURL(query string) bool {
	lower := strings.ToLower(query)
	return strings.Contains(lower, "youtube.com/") || strings.Contains(lower, "youtu.be/")
}

func isSpotifyURL(query string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(query)), "https://open.spotify.com/")
}

func isSoundCloudURL(query string) bool {
	lower := strings.ToLower(query)
	return strings.Contains(lower, "soundcloud.com/")
}
