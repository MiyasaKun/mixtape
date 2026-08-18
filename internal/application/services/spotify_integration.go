package services

import (
	"fiber_test/internal/adapters/outbound/postgres"
	"fiber_test/internal/adapters/outbound/spotify"
)

type SpotifyIntegrationService struct {
	spotify *spotify.Spotify
	db      *postgres.Postgres
}

func NewSpotifyIntegrationService(sp *spotify.Spotify, db *postgres.Postgres) *SpotifyIntegrationService {
	return &SpotifyIntegrationService{
		spotify: sp,
		db:      db,
	}
}
