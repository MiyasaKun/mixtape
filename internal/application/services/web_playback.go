package services

import (
	"context"
	"errors"
	"fmt"
)

// ErrNoPlayableMatch means a Spotify track link was resolved, but YouTube
// returned no candidate to play.
var ErrNoPlayableMatch = errors.New("no playable match found")

// SpotifyCandidateResolver turns a Spotify track link into playable YouTube
// candidates plus the Spotify cover art.
type SpotifyCandidateResolver interface {
	SpotifyYouTubeOptions(ctx context.Context, spotifyURL string) ([]Track, string, error)
}

// PlaybackStarter adds a resolved track to a guild queue and starts playback
// when the guild is idle.
type PlaybackStarter interface {
	PlayWithCover(ctx context.Context, guildID, query, coverURL string) (*PlaybackResult, error)
}

// WebPlaybackService sends a Spotify track chosen in the web UI to the bot.
// The Discord flow lets people pick a YouTube candidate; the web flow has no
// such step, so it uses the best match.
type WebPlaybackService struct {
	resolver SpotifyCandidateResolver
	playback PlaybackStarter
}

func NewWebPlaybackService(resolver SpotifyCandidateResolver, playback PlaybackStarter) *WebPlaybackService {
	return &WebPlaybackService{resolver: resolver, playback: playback}
}

// EnqueueSpotify resolves a Spotify track link to a YouTube match and hands it
// to playback, which queues it and starts it when nothing is playing.
func (s *WebPlaybackService) EnqueueSpotify(ctx context.Context, guildID, spotifyURL string) (*PlaybackResult, error) {
	candidates, coverURL, err := s.resolver.SpotifyYouTubeOptions(ctx, spotifyURL)
	if err != nil {
		return nil, fmt.Errorf("resolve spotify track: %w", err)
	}
	if len(candidates) == 0 {
		return nil, ErrNoPlayableMatch
	}
	return s.playback.PlayWithCover(ctx, guildID, candidates[0].URL, coverURL)
}
