package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"fiber_test/internal/application/services"
	"fiber_test/internal/domain"
)

func connectedStore(userID string, expiry time.Time) *fakeSpotifyStore {
	return &fakeSpotifyStore{conns: map[string]domain.SpotifyConnection{
		userID: {
			UserID:             userID,
			SpotifyUserID:      "sp-1",
			AccessToken:        "stored-access",
			RefreshTokenCipher: "enc:stored-refresh",
			TokenExpiry:        expiry,
		},
	}}
}

func TestListPlaylistsFailsWhenSpotifyNotConnected(t *testing.T) {
	store := &fakeSpotifyStore{}
	provider := &fakeSpotifyProvider{}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	_, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50)
	if !errors.Is(err, domain.ErrSpotifyNotConnected) {
		t.Fatalf("want ErrSpotifyNotConnected, got %v", err)
	}
	if provider.refreshCalls != 0 {
		t.Error("no refresh should happen without a connection")
	}
}

func TestListPlaylistsUsesFreshAccessToken(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(time.Hour))
	provider := &fakeSpotifyProvider{
		playlists: domain.PlaylistPage{Items: []domain.Playlist{{ID: "pl-1", Name: "Focus", Browseable: true}}},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	page, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "pl-1" {
		t.Errorf("unexpected page: %+v", page)
	}
	if provider.refreshCalls != 0 {
		t.Error("a fresh token must not be refreshed")
	}
	if provider.gotListToken != "stored-access" {
		t.Errorf("want stored access token, got %q", provider.gotListToken)
	}
	if provider.gotListUserID != "sp-1" {
		t.Errorf("want spotify user id forwarded, got %q", provider.gotListUserID)
	}
}

func TestListPlaylistsRefreshesExpiredTokenAndPersistsIt(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(-time.Minute))
	provider := &fakeSpotifyProvider{
		refreshTokens: domain.SpotifyTokens{AccessToken: "fresh-access", Expiry: baseTime.Add(time.Hour)},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	if _, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.refreshCalls != 1 {
		t.Fatalf("want exactly one refresh, got %d", provider.refreshCalls)
	}
	if provider.gotRefresh != "stored-refresh" {
		t.Errorf("want decrypted refresh token, got %q", provider.gotRefresh)
	}
	if provider.gotListToken != "fresh-access" {
		t.Errorf("want refreshed access token used, got %q", provider.gotListToken)
	}

	saved := store.conns["u-1"]
	if saved.AccessToken != "fresh-access" {
		t.Errorf("want persisted fresh token, got %q", saved.AccessToken)
	}
	if !saved.TokenExpiry.Equal(baseTime.Add(time.Hour)) {
		t.Errorf("want persisted expiry, got %v", saved.TokenExpiry)
	}
	if saved.RefreshTokenCipher != "enc:stored-refresh" {
		t.Errorf("refresh token must be preserved when rotation is omitted, got %q", saved.RefreshTokenCipher)
	}
}

func TestListPlaylistsRefreshesTokenInsideSkewWindow(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(30*time.Second))
	provider := &fakeSpotifyProvider{
		refreshTokens: domain.SpotifyTokens{AccessToken: "fresh-access", Expiry: baseTime.Add(time.Hour)},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	if _, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.refreshCalls != 1 {
		t.Errorf("a token expiring within the skew window must refresh, got %d calls", provider.refreshCalls)
	}
}

func TestListPlaylistsPersistsRotatedRefreshToken(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(-time.Minute))
	provider := &fakeSpotifyProvider{
		refreshTokens: domain.SpotifyTokens{AccessToken: "fresh", RefreshToken: "rotated", Expiry: baseTime.Add(time.Hour)},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	if _, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved := store.conns["u-1"]; saved.RefreshTokenCipher != "enc:rotated" {
		t.Errorf("want rotated refresh token encrypted, got %q", saved.RefreshTokenCipher)
	}
}

func TestListPlaylistsRequiresReconnectWhenRefreshRejected(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(-time.Minute))
	provider := &fakeSpotifyProvider{refreshErr: domain.ErrSpotifyReconnectRequired}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	_, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50)
	if !errors.Is(err, domain.ErrSpotifyReconnectRequired) {
		t.Fatalf("want ErrSpotifyReconnectRequired, got %v", err)
	}
	if saved := store.conns["u-1"]; saved.AccessToken != "stored-access" {
		t.Error("a rejected refresh must not overwrite the stored connection")
	}
}

func TestListPlaylistsHidesUnbrowseablePlaylists(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(time.Hour))
	provider := &fakeSpotifyProvider{
		playlists: domain.PlaylistPage{Items: []domain.Playlist{
			{ID: "pl-owned", Name: "Owned", Browseable: true},
			{ID: "pl-followed", Name: "Followed", Browseable: false},
		}},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	page, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "pl-owned" {
		t.Fatalf("want only the browseable playlist, got %+v", page.Items)
	}
}

func TestListPlaylistsSkipsPagesWithOnlyUnbrowseablePlaylists(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(time.Hour))
	provider := &fakeSpotifyProvider{
		playlistPages: map[int]domain.PlaylistPage{
			0: {Items: []domain.Playlist{
				{ID: "pl-hidden-1", Browseable: false},
				{ID: "pl-hidden-2", Browseable: false},
			}, Next: intPtr(2)},
			2: {Items: []domain.Playlist{{
				ID: "pl-visible", Name: "Visible", Browseable: true,
			}}},
		},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	page, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "pl-visible" {
		t.Fatalf("want playlist from the second page, got %+v", page.Items)
	}
	if page.Next != nil {
		t.Fatalf("want exhausted page, got next %d", *page.Next)
	}
	if provider.gotListOffset != 2 {
		t.Errorf("want second page fetched at offset 2, got %d", provider.gotListOffset)
	}
}

func intPtr(value int) *int { return &value }

func TestListPlaylistsCapsSkippedHiddenPages(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(time.Hour))
	calls := 0
	provider := &fakeSpotifyProvider{
		// An upstream that always reports hidden playlists on an ever-
		// advancing offset must not be followed forever.
		playlistFn: func(offset int) (domain.PlaylistPage, error) {
			calls++
			next := offset + 1
			return domain.PlaylistPage{
				Items: []domain.Playlist{{ID: "hidden", Browseable: false}},
				Next:  &next,
			}, nil
		},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	page, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("want an empty visible page, got %+v", page.Items)
	}
	if page.Next == nil || *page.Next != 10 {
		t.Fatalf("want the furthest known offset as next, got %v", page.Next)
	}
	if calls != 10 {
		t.Errorf("want the page fetch loop capped at 10, got %d calls", calls)
	}
}

func TestListLikedTracksUsesStoredToken(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(time.Hour))
	provider := &fakeSpotifyProvider{
		liked: domain.TrackPage{Items: []domain.PlaylistTrack{{ID: "tr-1", Title: "Nightcall"}}},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	page, err := svc.ListLikedTracks(context.Background(), "u-1", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Title != "Nightcall" {
		t.Errorf("unexpected page: %+v", page)
	}
	if provider.gotLikedToken != "stored-access" {
		t.Errorf("want stored access token, got %q", provider.gotLikedToken)
	}
}

func TestListPlaylistTracksForwardsPlaylistID(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(time.Hour))
	provider := &fakeSpotifyProvider{
		tracks: domain.TrackPage{Items: []domain.PlaylistTrack{{ID: "tr-9", Title: "Resonance"}}},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	page, err := svc.ListPlaylistTracks(context.Background(), "u-1", "pl-9", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "tr-9" {
		t.Errorf("unexpected page: %+v", page)
	}
	if provider.gotPlaylistID != "pl-9" {
		t.Errorf("want playlist id pl-9, got %q", provider.gotPlaylistID)
	}
}

func TestListPlaylistsDoesNotRefreshTwiceForFreshToken(t *testing.T) {
	store := connectedStore("u-1", baseTime.Add(-time.Minute))
	provider := &fakeSpotifyProvider{
		refreshTokens: domain.SpotifyTokens{AccessToken: "fresh", Expiry: baseTime.Add(time.Hour)},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	if _, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The refreshed token was persisted, so a second read needs no refresh.
	if _, err := svc.ListPlaylists(context.Background(), "u-1", 0, 50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.refreshCalls != 1 {
		t.Errorf("want a single refresh across calls, got %d", provider.refreshCalls)
	}
}
