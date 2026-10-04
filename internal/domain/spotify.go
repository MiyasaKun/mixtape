package domain

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for the Spotify integration.
var (
	// ErrSpotifyNotConnected means the user has not linked a Spotify account.
	ErrSpotifyNotConnected = errors.New("spotify not connected")
	// ErrSpotifyReconnectRequired means Spotify rejected the stored refresh
	// token; the user must connect again.
	ErrSpotifyReconnectRequired = errors.New("spotify reconnect required")
)

// SpotifyAPIError is a non-success response from the Spotify Web API. It
// carries the upstream status so callers can react to specific conditions
// (for example 403 when a playlist's contents are not readable).
type SpotifyAPIError struct {
	StatusCode int
	Operation  string
	Body       string
}

func (e *SpotifyAPIError) Error() string {
	return fmt.Sprintf("spotify %s failed: status %d: %s", e.Operation, e.StatusCode, e.Body)
}

// SpotifyTokens is the OAuth token pair returned by Spotify's token endpoint.
type SpotifyTokens struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

// SpotifyProfile is the Spotify account linked to a Mixtape user.
type SpotifyProfile struct {
	ID          string
	DisplayName string
}
