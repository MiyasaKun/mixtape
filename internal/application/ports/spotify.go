package ports

import (
	"context"

	"fiber_test/internal/domain"
)

// SpotifyConnectionStore persists the optional Spotify link for a Mixtape
// user. Refresh tokens are stored encrypted.
type SpotifyConnectionStore interface {
	Save(ctx context.Context, conn domain.SpotifyConnection) error
	ByUserID(ctx context.Context, userID string) (domain.SpotifyConnection, error)
	Delete(ctx context.Context, userID string) error
}

// SpotifyProvider performs Spotify's authorization-code exchange and describes
// its authorization endpoint. It also reads the signed-in user's playlists and
// liked songs and refreshes expired access tokens.
type SpotifyProvider interface {
	ExchangeCode(ctx context.Context, code, redirectURI string) (domain.SpotifyTokens, domain.SpotifyProfile, error)
	AuthURL(state string) string
	// RefreshAccessToken trades a refresh token for a fresh access token.
	// It returns domain.ErrSpotifyReconnectRequired when Spotify rejects the
	// refresh token (for example invalid_grant).
	RefreshAccessToken(ctx context.Context, refreshToken string) (domain.SpotifyTokens, error)
	// ListPlaylists returns a page of the user's followed and owned playlists.
	// spotifyUserID marks which playlists the user owns.
	ListPlaylists(ctx context.Context, accessToken, spotifyUserID string, offset, limit int) (domain.PlaylistPage, error)
	// ListLikedTracks returns a page of the user's Liked Songs.
	ListLikedTracks(ctx context.Context, accessToken string, offset, limit int) (domain.TrackPage, error)
	// ListPlaylistTracks returns a page of tracks in a single playlist.
	ListPlaylistTracks(ctx context.Context, accessToken, playlistID string, offset, limit int) (domain.TrackPage, error)
}

// Encryptor protects secrets at rest. Implementations must return ciphertext
// that only Decrypt can read back.
type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}
