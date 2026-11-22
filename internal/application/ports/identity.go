package ports

import (
	"context"
	"time"

	"fiber_test/internal/domain"
)

// UserRepository persists Mixtape accounts keyed by their external Google
// identity.
type UserRepository interface {
	// UpsertByGoogleID creates the account on first sign-in and refreshes
	// profile fields on later sign-ins.
	UpsertByGoogleID(ctx context.Context, profile domain.GoogleProfile) (domain.User, error)
	// ByID looks an account up by its internal ID.
	ByID(ctx context.Context, id string) (domain.User, error)
	// UpdateSpotifyAutoplay stores whether Spotify links should autoplay.
	UpdateSpotifyAutoplay(ctx context.Context, id string, enabled bool) error
}

// SessionRepository manages sign-in sessions in PostgreSQL.
type SessionRepository interface {
	// Create issues a new session for the user with the given time-to-live.
	Create(ctx context.Context, userID string, ttl time.Duration) (domain.Session, error)
	// Get returns the session with the given ID.
	Get(ctx context.Context, id string) (domain.Session, error)
	// Delete revokes one session.
	Delete(ctx context.Context, id string) error
	// DeleteExpired removes sessions past their expiry and returns how many
	// rows were removed.
	DeleteExpired(ctx context.Context) (int64, error)
}

// GoogleIdentityProvider performs the Google OAuth authorization-code exchange
// and knows how to build Google authorization and sign-out URLs.
type GoogleIdentityProvider interface {
	// AuthURL returns the Google authorization endpoint for consent.
	AuthURL(state string) string
	// ExchangeCode trades an OAuth authorization code for the Google
	// profile of the person who approved it.
	ExchangeCode(ctx context.Context, code, redirectURI string) (domain.GoogleProfile, error)
	// SignOutURL returns the Google logout endpoint for post-sign-out
	// cleanup.
	SignOutURL(redirectURI string) string
}
