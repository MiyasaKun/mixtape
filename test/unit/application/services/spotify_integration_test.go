package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"fiber_test/internal/application/services"
	"fiber_test/internal/domain"
)

type fakeSpotifyStore struct {
	conns   map[string]domain.SpotifyConnection
	deleted []string
}

func (f *fakeSpotifyStore) Save(ctx context.Context, c domain.SpotifyConnection) error {
	if f.conns == nil {
		f.conns = map[string]domain.SpotifyConnection{}
	}
	f.conns[c.UserID] = c
	return nil
}

func (f *fakeSpotifyStore) ByUserID(ctx context.Context, userID string) (domain.SpotifyConnection, error) {
	c, ok := f.conns[userID]
	if !ok {
		return domain.SpotifyConnection{}, domain.ErrUserNotFound
	}
	return c, nil
}

func (f *fakeSpotifyStore) Delete(ctx context.Context, userID string) error {
	f.deleted = append(f.deleted, userID)
	delete(f.conns, userID)
	return nil
}

type fakeSpotifyProvider struct {
	tokens  domain.SpotifyTokens
	profile domain.SpotifyProfile
	err     error
	gotCode string

	refreshTokens domain.SpotifyTokens
	refreshErr    error
	gotRefresh    string
	refreshCalls  int

	playlists      domain.PlaylistPage
	playlistPages  map[int]domain.PlaylistPage
	playlistFn     func(offset int) (domain.PlaylistPage, error)
	liked          domain.TrackPage
	tracks         domain.TrackPage
	listErr        error
	gotListToken   string
	gotListUserID  string
	gotListOffset  int
	gotLikedToken  string
	gotTracksToken string
	gotPlaylistID  string
}

func (f *fakeSpotifyProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (domain.SpotifyTokens, domain.SpotifyProfile, error) {
	f.gotCode = code
	return f.tokens, f.profile, f.err
}

func (f *fakeSpotifyProvider) AuthURL(state string) string {
	return "https://accounts.spotify.com/authorize?state=" + state
}

func (f *fakeSpotifyProvider) RefreshAccessToken(ctx context.Context, refreshToken string) (domain.SpotifyTokens, error) {
	f.refreshCalls++
	f.gotRefresh = refreshToken
	return f.refreshTokens, f.refreshErr
}

func (f *fakeSpotifyProvider) ListPlaylists(ctx context.Context, accessToken, spotifyUserID string, offset, limit int) (domain.PlaylistPage, error) {
	f.gotListToken = accessToken
	f.gotListUserID = spotifyUserID
	f.gotListOffset = offset
	if f.playlistPages != nil {
		return f.playlistPages[offset], f.listErr
	}
	if f.playlistFn != nil {
		return f.playlistFn(offset)
	}
	return f.playlists, f.listErr
}

func (f *fakeSpotifyProvider) ListLikedTracks(ctx context.Context, accessToken string, offset, limit int) (domain.TrackPage, error) {
	f.gotLikedToken = accessToken
	return f.liked, f.listErr
}

func (f *fakeSpotifyProvider) ListPlaylistTracks(ctx context.Context, accessToken, playlistID string, offset, limit int) (domain.TrackPage, error) {
	f.gotTracksToken = accessToken
	f.gotPlaylistID = playlistID
	return f.tracks, f.listErr
}

type fakeEncryptor struct{}

func (fakeEncryptor) Encrypt(plain string) (string, error) { return "enc:" + plain, nil }
func (fakeEncryptor) Decrypt(cipher string) (string, error) {
	return cipher[len("enc:"):], nil
}

func TestConnectSpotifyEncryptsRefreshToken(t *testing.T) {
	store := &fakeSpotifyStore{}
	provider := &fakeSpotifyProvider{
		tokens:  domain.SpotifyTokens{AccessToken: "access", RefreshToken: "refresh-secret", Expiry: baseTime.Add(time.Hour)},
		profile: domain.SpotifyProfile{ID: "sp-1", DisplayName: "Listener"},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	if err := svc.Connect(context.Background(), "u-1", "code-1", "http://localhost:3000/api/auth/spotify/callback"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	saved, _ := store.ByUserID(context.Background(), "u-1")
	if saved.RefreshTokenCipher == "refresh-secret" {
		t.Fatal("refresh token must not be stored in plaintext")
	}
	if saved.RefreshTokenCipher != "enc:refresh-secret" {
		t.Errorf("want encrypted refresh token, got %q", saved.RefreshTokenCipher)
	}
	if saved.SpotifyUserID != "sp-1" {
		t.Errorf("want spotify user sp-1, got %q", saved.SpotifyUserID)
	}
}

func TestConnectSpotifyPropagatesProviderError(t *testing.T) {
	store := &fakeSpotifyStore{}
	provider := &fakeSpotifyProvider{err: errors.New("invalid code")}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	if err := svc.Connect(context.Background(), "u-1", "bad", "r"); err == nil {
		t.Fatal("want provider error to propagate")
	}
	if len(store.conns) != 0 {
		t.Error("no connection should be stored when exchange fails")
	}
}

func TestDisconnectSpotifyRemovesConnection(t *testing.T) {
	store := &fakeSpotifyStore{}
	provider := &fakeSpotifyProvider{
		tokens:  domain.SpotifyTokens{RefreshToken: "refresh"},
		profile: domain.SpotifyProfile{ID: "sp-1"},
	}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	if err := svc.Connect(context.Background(), "u-1", "code", "r"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := svc.Disconnect(context.Background(), "u-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.deleted) != 1 || store.deleted[0] != "u-1" {
		t.Errorf("want delete for u-1, got %v", store.deleted)
	}
}

func TestSpotifyStatusReflectsConnection(t *testing.T) {
	store := &fakeSpotifyStore{}
	provider := &fakeSpotifyProvider{tokens: domain.SpotifyTokens{RefreshToken: "r"}, profile: domain.SpotifyProfile{ID: "sp-1", DisplayName: "Listener"}}
	svc := services.NewSpotifyIntegrationService(provider, store, fakeEncryptor{}, fakeNow)

	if connected, _ := svc.Status(context.Background(), "u-1"); connected {
		t.Error("want disconnected before connect")
	}

	if err := svc.Connect(context.Background(), "u-1", "code", "r"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	connected, err := svc.Status(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !connected {
		t.Error("want connected after connect")
	}
}
