package domain_test

import (
	"testing"
	"time"

	"fiber_test/internal/domain"
)

func TestUserHasIdentityFields(t *testing.T) {
	created := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	user := domain.User{
		ID:        "user-1",
		GoogleID:  "google-1",
		Email:     "listener@example.com",
		Name:      "Listener",
		Locale:    "en",
		CreatedAt: created,
		UpdatedAt: created,
	}

	if user.GoogleID != "google-1" {
		t.Errorf("want GoogleID google-1, got %q", user.GoogleID)
	}
	if user.Email != "listener@example.com" {
		t.Errorf("want email listener@example.com, got %q", user.Email)
	}
	if !user.CreatedAt.Equal(created) {
		t.Errorf("want CreatedAt %v, got %v", created, user.CreatedAt)
	}
}

func TestSpotifyConnectionStoresCipherTextOnly(t *testing.T) {
	expires := time.Date(2026, 10, 4, 13, 0, 0, 0, time.UTC)
	conn := domain.SpotifyConnection{
		UserID:             "user-1",
		RefreshTokenCipher: "cipher-text",
		TokenExpiry:        expires,
	}

	if conn.RefreshTokenCipher != "cipher-text" {
		t.Errorf("RefreshTokenCipher should be set")
	}
	if !conn.TokenExpiry.Equal(expires) {
		t.Errorf("want expiry %v, got %v", expires, conn.TokenExpiry)
	}
}
