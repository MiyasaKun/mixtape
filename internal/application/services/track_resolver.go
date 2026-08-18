package services

import (
	"fiber_test/internal/adapters/outbound/soundcloud"
	"fiber_test/internal/adapters/outbound/spotify"
	"fiber_test/internal/adapters/outbound/youtube"
)

type TrackResolverService struct {
	youtube    *youtube.YouTube
	spotify    *spotify.Spotify
	soundcloud *soundcloud.SoundCloud
}

func NewTrackResolverService(yt *youtube.YouTube, sp *spotify.Spotify, sc *soundcloud.SoundCloud) *TrackResolverService {
	return &TrackResolverService{
		youtube:    yt,
		spotify:    sp,
		soundcloud: sc,
	}
}
