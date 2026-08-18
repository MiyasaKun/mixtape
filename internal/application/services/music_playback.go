package services

import (
	"context"
)

type MusicPlaybackService struct {
	queue         *QueueService
	trackResolver *TrackResolverService
}

func NewMusicPlaybackService(queue *QueueService, trackResolver *TrackResolverService) *MusicPlaybackService {
	return &MusicPlaybackService{
		queue:         queue,
		trackResolver: trackResolver,
	}
}

func (s *MusicPlaybackService) Play(ctx context.Context, guildID string, query string) error {
	return nil
}

func (s *MusicPlaybackService) Pause(ctx context.Context, guildID string) error {
	return nil
}

func (s *MusicPlaybackService) Resume(ctx context.Context, guildID string) error {
	return nil
}

func (s *MusicPlaybackService) Skip(ctx context.Context, guildID string) error {
	return nil
}

func (s *MusicPlaybackService) Stop(ctx context.Context, guildID string) error {
	return nil
}

func (s *MusicPlaybackService) Seek(ctx context.Context, guildID string, position int64) error {
	return nil
}

func (s *MusicPlaybackService) SetVolume(ctx context.Context, guildID string, volume int) error {
	return nil
}

func (s *MusicPlaybackService) GetCurrentTrack(ctx context.Context, guildID string) (*Track, error) {
	return nil, nil
}

type Track struct {
	ID           string
	Title        string
	Artist       string
	Duration     int64
	Source       string
	URL          string
	ThumbnailURL string
}
