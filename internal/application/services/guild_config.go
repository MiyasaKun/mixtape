package services

import (
	"context"

	"fiber_test/internal/adapters/outbound/postgres"
)

type GuildConfigService struct {
	db *postgres.Postgres
}

func NewGuildConfigService(db *postgres.Postgres) *GuildConfigService {
	return &GuildConfigService{
		db: db,
	}
}

type GuildConfig struct {
	GuildID        string
	TextChannelID  string
	VoiceChannelID string
	DefaultVolume  int
	AutoPlay       bool
	DJRoleID       string
}

func (s *GuildConfigService) GetConfig(ctx context.Context, guildID string) (*GuildConfig, error) {
	return nil, nil
}

func (s *GuildConfigService) SetVoiceChannel(ctx context.Context, guildID string, channelID string) error {
	return nil
}

func (s *GuildConfigService) SetTextChannel(ctx context.Context, guildID string, channelID string) error {
	return nil
}

func (s *GuildConfigService) SetDefaultVolume(ctx context.Context, guildID string, volume int) error {
	return nil
}

func (s *GuildConfigService) SetAutoPlay(ctx context.Context, guildID string, enabled bool) error {
	return nil
}

func (s *GuildConfigService) SetDJRole(ctx context.Context, guildID string, roleID string) error {
	return nil
}
