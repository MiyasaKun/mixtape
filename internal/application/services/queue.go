package services

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"sync"

	"fiber_test/internal/adapters/outbound/redis"

	redispkg "github.com/redis/go-redis/v9"
)

// queueKeyPrefix is the Redis key namespace for guild queues.
const queueKeyPrefix = "mixtape:queue:"

// QueueService manages per-guild play queues, persisted as JSON blobs in
// Redis. Each guild's queue is keyed by mixtape:queue:{guildID}. A mutex
// serializes read-modify-write operations so concurrent updates from
// commands and the playback loop cannot clobber each other.
type QueueService struct {
	cache *redis.Redis

	mu sync.Mutex
}

// NewQueueService builds a queue service backed by the given Redis cache.
func NewQueueService(cache *redis.Redis) *QueueService {
	return &QueueService{
		cache: cache,
	}
}

// Queue is the play queue of a single guild. It only contains tracks that
// have not been played yet: the playback service pops the front track when it
// starts playing, so already-played tracks are removed from the queue.
type Queue struct {
	GuildID  string
	Tracks   []*Track
	LoopMode string
}

// GetQueue returns the queue for a guild, or nil if none exists yet.
func (s *QueueService) GetQueue(ctx context.Context, guildID string) (*Queue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.getQueue(ctx, guildID)
}

// AddToQueue appends a track to the guild's queue, creating the queue if it
// does not exist yet.
func (s *QueueService) AddToQueue(ctx context.Context, guildID string, track *Track) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	queue, err := s.getQueue(ctx, guildID)
	if err != nil {
		return err
	}
	if queue == nil {
		queue = &Queue{GuildID: guildID}
	}
	queue.Tracks = append(queue.Tracks, track)

	return s.saveQueue(ctx, queue)
}

// RemoveFromQueue removes the track at the given index.
func (s *QueueService) RemoveFromQueue(ctx context.Context, guildID string, index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	queue, err := s.getQueue(ctx, guildID)
	if err != nil {
		return err
	}
	if queue == nil || index < 0 || index >= len(queue.Tracks) {
		return errors.New("invalid queue index")
	}

	queue.Tracks = append(queue.Tracks[:index], queue.Tracks[index+1:]...)

	return s.saveQueue(ctx, queue)
}

// ClearQueue removes the entire queue for a guild.
func (s *QueueService) ClearQueue(ctx context.Context, guildID string) error {
	return s.cache.Client().Del(ctx, queueKey(guildID)).Err()
}

// ShuffleQueue randomly reorders the tracks in a guild's queue.
func (s *QueueService) ShuffleQueue(ctx context.Context, guildID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	queue, err := s.getQueue(ctx, guildID)
	if err != nil {
		return err
	}
	if queue == nil {
		return nil
	}

	rand.Shuffle(len(queue.Tracks), func(i, j int) {
		queue.Tracks[i], queue.Tracks[j] = queue.Tracks[j], queue.Tracks[i]
	})

	return s.saveQueue(ctx, queue)
}

// SetLoopMode persists the loop mode ("all", "one", or "") for a guild.
// Playback itself does not honor the mode yet.
func (s *QueueService) SetLoopMode(ctx context.Context, guildID string, mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	queue, err := s.getQueue(ctx, guildID)
	if err != nil {
		return err
	}
	if queue == nil {
		queue = &Queue{GuildID: guildID}
	}
	queue.LoopMode = mode

	return s.saveQueue(ctx, queue)
}

// getQueue reads and decodes the queue for a guild. Callers must hold s.mu.
func (s *QueueService) getQueue(ctx context.Context, guildID string) (*Queue, error) {
	raw, err := s.cache.Client().Get(ctx, queueKey(guildID)).Bytes()
	if err != nil {
		if errors.Is(err, redispkg.Nil) {
			return nil, nil
		}
		return nil, err
	}

	var queue Queue
	if err := json.Unmarshal(raw, &queue); err != nil {
		return nil, err
	}
	return &queue, nil
}

// saveQueue encodes and stores the queue. Callers must hold s.mu.
func (s *QueueService) saveQueue(ctx context.Context, queue *Queue) error {
	raw, err := json.Marshal(queue)
	if err != nil {
		return err
	}
	return s.cache.Client().Set(ctx, queueKey(queue.GuildID), raw, 0).Err()
}

func queueKey(guildID string) string {
	return queueKeyPrefix + guildID
}
