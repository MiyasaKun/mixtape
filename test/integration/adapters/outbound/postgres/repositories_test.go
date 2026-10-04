//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"fiber_test/internal/adapters/outbound/postgres"
	"fiber_test/internal/domain"
	"fiber_test/internal/infrastructure/logging"
)

func testDB(t *testing.T) *postgres.Postgres {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://user:pass@localhost:5432/discord_bot"
	}
	db, err := postgres.New(url, logging.NewLogger("test"), "test")
	if err != nil {
		t.Skipf("PostgreSQL is unavailable at %s: %v", url, err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestUserRepositoryCreatesOnFirstSignIn(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	repo := postgres.NewUserRepository(db)
	googleID := "integration-" + time.Now().Format(time.RFC3339Nano)

	first, err := repo.UpsertByGoogleID(ctx, domain.GoogleProfile{GoogleID: googleID, Email: "one@example.com", Name: "One", Picture: "https://example.com/one.jpg"})
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if first.ID == "" {
		t.Fatal("want a generated user ID")
	}

	second, err := repo.UpsertByGoogleID(ctx, domain.GoogleProfile{GoogleID: googleID, Email: "two@example.com", Name: "Two", Picture: "https://example.com/two.jpg"})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("want the same account, got %q then %q", first.ID, second.ID)
	}
	if second.Email != "two@example.com" {
		t.Errorf("want refreshed email, got %q", second.Email)
	}
	if second.Picture != "https://example.com/two.jpg" {
		t.Errorf("want refreshed picture, got %q", second.Picture)
	}
}

func TestSessionsRepositoryRoundTrip(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	users := postgres.NewUserRepository(db)
	sessions := postgres.NewSessionsRepository(db)

	user, err := users.UpsertByGoogleID(ctx, domain.GoogleProfile{GoogleID: "integration-session-" + time.Now().Format(time.RFC3339Nano), Email: "session@example.com", Name: "Session"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	created, err := sessions.Create(ctx, user.ID, time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.ID == "" {
		t.Fatal("want a session ID")
	}

	got, err := sessions.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.UserID != user.ID {
		t.Errorf("want user %q, got %q", user.ID, got.UserID)
	}

	if err := sessions.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if _, err := sessions.Get(ctx, created.ID); err == nil {
		t.Error("want deleted session to be gone")
	}
}

func TestSpotifyRepositoryStoresEncryptedToken(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	users := postgres.NewUserRepository(db)
	spotify := postgres.NewSpotifyRepository(db)

	user, err := users.UpsertByGoogleID(ctx, domain.GoogleProfile{GoogleID: "integration-spotify-" + time.Now().Format(time.RFC3339Nano), Email: "spotify@example.com", Name: "Spotify"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { _ = spotify.Delete(context.Background(), user.ID) })

	conn := domain.SpotifyConnection{
		UserID:             user.ID,
		SpotifyUserID:      "sp-1",
		SpotifyDisplayName: "Listener",
		AccessToken:        "access",
		RefreshTokenCipher: "cipher-text",
		TokenExpiry:        time.Now().Add(time.Hour),
	}
	if err := spotify.Save(ctx, conn); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := spotify.ByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.RefreshTokenCipher != "cipher-text" {
		t.Errorf("want stored cipher, got %q", got.RefreshTokenCipher)
	}
	if !got.TokenExpiry.After(time.Now()) {
		t.Errorf("want future expiry, got %v", got.TokenExpiry)
	}

	// The user lookup joins the connection state for /api/auth/me.
	withConnection, err := users.ByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if !withConnection.SpotifyConnected {
		t.Error("want SpotifyConnected true once a connection exists")
	}
}
