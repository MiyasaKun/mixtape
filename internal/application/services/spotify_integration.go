package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"fiber_test/internal/application/ports"
	"fiber_test/internal/domain"
)

// tokenRefreshSkew refreshes an access token slightly before it actually
// expires so an in-flight request never races the expiry.
const tokenRefreshSkew = 60 * time.Second

// SpotifyIntegrationService links an optional Spotify account to a Mixtape
// user and reads their playlists and liked songs. Access tokens are
// short-lived; the refresh token is encrypted before it is persisted.
type SpotifyIntegrationService struct {
	provider  ports.SpotifyProvider
	store     ports.SpotifyConnectionStore
	encryptor ports.Encryptor
	now       func() time.Time
	refresh   sync.Map // userID -> *sync.Mutex, single-flight guard
}

func NewSpotifyIntegrationService(
	provider ports.SpotifyProvider,
	store ports.SpotifyConnectionStore,
	encryptor ports.Encryptor,
	now func() time.Time,
) *SpotifyIntegrationService {
	return &SpotifyIntegrationService{
		provider:  provider,
		store:     store,
		encryptor: encryptor,
		now:       now,
	}
}

// AuthURL builds the Spotify authorization URL for a given anti-CSRF state.
func (s *SpotifyIntegrationService) AuthURL(state string) string {
	return s.provider.AuthURL(state)
}

// Connect exchanges the authorization code and stores the encrypted Spotify
// link for the user.
func (s *SpotifyIntegrationService) Connect(ctx context.Context, userID, code, redirectURI string) error {
	tokens, profile, err := s.provider.ExchangeCode(ctx, code, redirectURI)
	if err != nil {
		return err
	}

	refreshCipher, err := s.encryptor.Encrypt(tokens.RefreshToken)
	if err != nil {
		return err
	}

	return s.store.Save(ctx, domain.SpotifyConnection{
		UserID:             userID,
		SpotifyUserID:      profile.ID,
		SpotifyDisplayName: profile.DisplayName,
		AccessToken:        tokens.AccessToken,
		RefreshTokenCipher: refreshCipher,
		TokenExpiry:        tokens.Expiry,
	})
}

// Disconnect removes the user's Spotify link.
func (s *SpotifyIntegrationService) Disconnect(ctx context.Context, userID string) error {
	return s.store.Delete(ctx, userID)
}

// Status reports whether the user currently has a Spotify connection.
func (s *SpotifyIntegrationService) Status(ctx context.Context, userID string) (bool, error) {
	_, err := s.store.ByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListPlaylists returns a page of the user's playlists that Mixtape can
// actually browse. Playlists Spotify does not expose contents for are dropped
// here so they never reach the web client. When a whole upstream page is
// hidden the next page is fetched instead, keeping the visible list and the
// reported `next` offset consistent.
func (s *SpotifyIntegrationService) ListPlaylists(ctx context.Context, userID string, offset, limit int) (domain.PlaylistPage, error) {
	conn, err := s.liveConnection(ctx, userID)
	if err != nil {
		return domain.PlaylistPage{}, err
	}
	return s.browseablePlaylists(ctx, conn.AccessToken, conn.SpotifyUserID, offset, limit)
}

// maxBrowseablePageFetches bounds how many consecutive all-hidden pages are
// skipped in one request before the visible list is returned as-is.
const maxBrowseablePageFetches = 10

func (s *SpotifyIntegrationService) browseablePlaylists(ctx context.Context, accessToken, spotifyUserID string, offset, limit int) (domain.PlaylistPage, error) {
	next := offset
	for range maxBrowseablePageFetches {
		page, err := s.provider.ListPlaylists(ctx, accessToken, spotifyUserID, next, limit)
		if err != nil {
			return domain.PlaylistPage{}, err
		}

		visible := make([]domain.Playlist, 0, len(page.Items))
		for _, playlist := range page.Items {
			if playlist.Browseable {
				visible = append(visible, playlist)
			}
		}

		if len(visible) > 0 || page.Next == nil || *page.Next <= next {
			page.Items = visible
			return page, nil
		}
		next = *page.Next
	}

	// The upstream kept reporting hidden pages; surface an empty page with the
	// furthest known offset rather than looping indefinitely.
	return domain.PlaylistPage{Items: nil, Next: &next}, nil
}

// ListLikedTracks returns a page of the user's Liked Songs.
func (s *SpotifyIntegrationService) ListLikedTracks(ctx context.Context, userID string, offset, limit int) (domain.TrackPage, error) {
	conn, err := s.liveConnection(ctx, userID)
	if err != nil {
		return domain.TrackPage{}, err
	}
	return s.provider.ListLikedTracks(ctx, conn.AccessToken, offset, limit)
}

// ListPlaylistTracks returns a page of tracks in a single playlist.
func (s *SpotifyIntegrationService) ListPlaylistTracks(ctx context.Context, userID, playlistID string, offset, limit int) (domain.TrackPage, error) {
	conn, err := s.liveConnection(ctx, userID)
	if err != nil {
		return domain.TrackPage{}, err
	}
	return s.provider.ListPlaylistTracks(ctx, conn.AccessToken, playlistID, offset, limit)
}

// liveConnection loads the user's Spotify connection, refreshing the access
// token when it is expired or about to expire.
func (s *SpotifyIntegrationService) liveConnection(ctx context.Context, userID string) (domain.SpotifyConnection, error) {
	conn, err := s.connection(ctx, userID)
	if err != nil {
		return domain.SpotifyConnection{}, err
	}
	if s.isFresh(conn) {
		return conn, nil
	}
	return s.refreshConnection(ctx, userID)
}

func (s *SpotifyIntegrationService) connection(ctx context.Context, userID string) (domain.SpotifyConnection, error) {
	conn, err := s.store.ByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.SpotifyConnection{}, domain.ErrSpotifyNotConnected
		}
		return domain.SpotifyConnection{}, err
	}
	return conn, nil
}

func (s *SpotifyIntegrationService) isFresh(conn domain.SpotifyConnection) bool {
	return s.now().Add(tokenRefreshSkew).Before(conn.TokenExpiry)
}

// refreshConnection refreshes the token once per user even under concurrent
// requests: the connection is re-read under a per-user lock and returned as-is
// if another caller already refreshed it.
func (s *SpotifyIntegrationService) refreshConnection(ctx context.Context, userID string) (domain.SpotifyConnection, error) {
	value, _ := s.refresh.LoadOrStore(userID, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	conn, err := s.connection(ctx, userID)
	if err != nil {
		return domain.SpotifyConnection{}, err
	}
	if s.isFresh(conn) {
		return conn, nil
	}

	refreshToken, err := s.encryptor.Decrypt(conn.RefreshTokenCipher)
	if err != nil {
		return domain.SpotifyConnection{}, fmt.Errorf("decrypt refresh token: %w", err)
	}

	tokens, err := s.provider.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		return domain.SpotifyConnection{}, err
	}

	conn.AccessToken = tokens.AccessToken
	conn.TokenExpiry = tokens.Expiry
	if tokens.RefreshToken != "" {
		cipher, err := s.encryptor.Encrypt(tokens.RefreshToken)
		if err != nil {
			return domain.SpotifyConnection{}, err
		}
		conn.RefreshTokenCipher = cipher
	}

	if err := s.store.Save(ctx, conn); err != nil {
		return domain.SpotifyConnection{}, err
	}
	return conn, nil
}
