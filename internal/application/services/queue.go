package services

import (
	"context"

	"fiber_test/internal/adapters/outbound/redis"
)

type QueueService struct {
	cache *redis.Redis
}

func NewQueueService(cache *redis.Redis) *QueueService {
	return &QueueService{
		cache: cache,
	}
}

type Queue struct {
	GuildID      string
	Tracks       []*Track
	CurrentIndex int
	LoopMode     string
}

func (s *QueueService) GetQueue(ctx context.Context, guildID string) (*Queue, error) {
	return nil, nil
}

func (s *QueueService) AddToQueue(ctx context.Context, guildID string, track *Track) error {
	return nil
}

func (s *QueueService) RemoveFromQueue(ctx context.Context, guildID string, index int) error {
	return nil
}

func (s *QueueService) ClearQueue(ctx context.Context, guildID string) error {
	return nil
}

func (s *QueueService) ShuffleQueue(ctx context.Context, guildID string) error {
	return nil
}

func (s *QueueService) SetLoopMode(ctx context.Context, guildID string, mode string) error {
	return nil
}
