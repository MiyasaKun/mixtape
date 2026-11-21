package domain

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Sentinel errors shared across the auth domain.
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

// GoogleProfile is the identity information we rely on from Google sign-in.
type GoogleProfile struct {
	GoogleID string
	Email    string
	Name     string
	Picture  string
}

// User is a Mixtape account, identified by the external Google identity that
// created it on first sign-in.
type User struct {
	ID               string
	GoogleID         string
	Email            string
	Name             string
	Picture          string
	SpotifyConnected bool
	Locale           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Session represents a signed-in browser session, referenced by an opaque
// random ID stored in an HttpOnly cookie.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// SpotifyConnection is the optional Spotify link on top of a Mixtape user.
// RefreshTokenCipher is an app-encrypted token, never stored in plaintext.
type SpotifyConnection struct {
	UserID             string
	SpotifyUserID      string
	SpotifyDisplayName string
	AccessToken        string
	RefreshTokenCipher string
	TokenExpiry        time.Time
	gorm.Model
}
