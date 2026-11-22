package services_test

import (
	"context"
	"testing"
	"time"

	"fiber_test/internal/application/services"
	"fiber_test/internal/domain"
)

// --- fakes ---

type fakeUserRepo struct {
	users    map[string]domain.User // by GoogleID
	upserted []domain.User
}

func (f *fakeUserRepo) UpsertByGoogleID(ctx context.Context, profile domain.GoogleProfile) (domain.User, error) {
	f.upserted = append(f.upserted, domain.User{GoogleID: profile.GoogleID, Email: profile.Email, Name: profile.Name, Picture: profile.Picture})
	u, ok := f.users[profile.GoogleID]
	if !ok {
		u = domain.User{ID: "u-" + profile.GoogleID, GoogleID: profile.GoogleID, Email: profile.Email, Name: profile.Name, Picture: profile.Picture, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		f.users[profile.GoogleID] = u
	}
	return u, nil
}

func (f *fakeUserRepo) UpdateSpotifyAutoplay(ctx context.Context, id string, enabled bool) error {
	for googleID, user := range f.users {
		if user.ID == id {
			user.SpotifyAutoplay = enabled
			f.users[googleID] = user
			return nil
		}
	}
	return domain.ErrUserNotFound
}

func (f *fakeUserRepo) ByID(ctx context.Context, id string) (domain.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

type fakeSessionRepo struct {
	sessions map[string]domain.Session
	revoked  []string
	now      func() time.Time
}

func (f *fakeSessionRepo) Create(ctx context.Context, userID string, ttl time.Duration) (domain.Session, error) {
	if f.sessions == nil {
		f.sessions = map[string]domain.Session{}
	}
	s := domain.Session{ID: "sess-" + userID, UserID: userID, ExpiresAt: f.now().Add(ttl), CreatedAt: f.now()}
	f.sessions[s.ID] = s
	return s, nil
}

func (f *fakeSessionRepo) Get(ctx context.Context, id string) (domain.Session, error) {
	s, ok := f.sessions[id]
	if !ok {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	return s, nil
}

func (f *fakeSessionRepo) Delete(ctx context.Context, id string) error {
	f.revoked = append(f.revoked, id)
	delete(f.sessions, id)
	return nil
}

func (f *fakeSessionRepo) DeleteExpired(ctx context.Context) (int64, error) { return 0, nil }

type fakeIdentities struct {
	profile domain.GoogleProfile
	err     error
}

func (f *fakeIdentities) ExchangeCode(ctx context.Context, code, redirectURI string) (domain.GoogleProfile, error) {
	return f.profile, f.err
}

func (f *fakeIdentities) SignOutURL(redirectURI string) string {
	return "https://accounts.google.com/logout"
}

func (f *fakeIdentities) AuthURL(state string) string {
	return "https://accounts.google.com/o/oauth2/v2/auth?state=" + state
}

// --- tests ---

func newAuthService(repo *fakeUserRepo, sessions *fakeSessionRepo, identities *fakeIdentities) *services.AuthService {
	return services.NewAuthService(repo, sessions, identities, fakeNow, 30*24*time.Hour)
}

var baseTime = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
var fakeNow = func() time.Time { return baseTime }

func TestFirstSignInCreatesAccountAndSession(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]domain.User{}}
	sessions := &fakeSessionRepo{now: fakeNow}
	svc := newAuthService(repo, sessions, &fakeIdentities{profile: domain.GoogleProfile{
		GoogleID: "google-1", Email: "listener@example.com", Name: "Listener",
	}})

	session, err := svc.SignInWithGoogle(context.Background(), "auth-code", "http://localhost:3000/api/auth/google/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.UserID != "u-google-1" {
		t.Errorf("want session for u-google-1, got %q", session.UserID)
	}
	if len(repo.upserted) != 1 || repo.upserted[0].GoogleID != "google-1" {
		t.Errorf("expected one upsert for google-1, got %+v", repo.upserted)
	}
	if !session.ExpiresAt.Equal(baseTime.Add(30 * 24 * time.Hour)) {
		t.Errorf("want 30-day expiry, got %v", session.ExpiresAt)
	}
}

func TestReturningUserIsNotDuplicated(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]domain.User{}}
	sessions := &fakeSessionRepo{now: fakeNow}
	svc := newAuthService(repo, sessions, &fakeIdentities{profile: domain.GoogleProfile{
		GoogleID: "google-1", Email: "listener@example.com", Name: "Listener",
	}})

	if _, err := svc.SignInWithGoogle(context.Background(), "a", "r"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.SignInWithGoogle(context.Background(), "b", "r"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.upserted) != 2 {
		t.Fatalf("want two upserts (one per sign-in), got %d", len(repo.upserted))
	}
	if len(repo.users) != 1 {
		t.Errorf("want single account, got %d", len(repo.users))
	}
}

func TestExpiredSessionIsRejected(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]domain.User{}}
	sessions := &fakeSessionRepo{now: fakeNow}
	svc := newAuthService(repo, sessions, &fakeIdentities{})

	s, err := sessions.Create(context.Background(), "u-1", 30*24*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Simulate the clock moving past expiry.
	sessions.sessions[s.ID] = domain.Session{ID: s.ID, UserID: s.UserID, CreatedAt: s.CreatedAt, ExpiresAt: baseTime.Add(-time.Second)}

	if _, err := svc.ValidateSession(context.Background(), s.ID); err == nil {
		t.Error("want expired session to be rejected")
	}
}

func TestSignOutRevokesSession(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]domain.User{}}
	sessions := &fakeSessionRepo{now: fakeNow}
	svc := newAuthService(repo, sessions, &fakeIdentities{})

	s, _ := sessions.Create(context.Background(), "u-1", 30*24*time.Hour)
	if err := svc.SignOut(context.Background(), s.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := svc.ValidateSession(context.Background(), s.ID); err == nil {
		t.Error("want revoked session to be rejected")
	}
	if len(sessions.revoked) != 1 || sessions.revoked[0] != s.ID {
		t.Errorf("want Delete called for session %q, got %v", s.ID, sessions.revoked)
	}
}

func TestCurrentUserResolvesSessionOwner(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]domain.User{
		"google-1": {ID: "u-google-1", GoogleID: "google-1", Email: "listener@example.com", Name: "Listener"},
	}}
	sessions := &fakeSessionRepo{now: fakeNow}
	svc := newAuthService(repo, sessions, &fakeIdentities{})

	s, _ := sessions.Create(context.Background(), "u-google-1", 30*24*time.Hour)

	user, err := svc.CurrentUser(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "u-google-1" {
		t.Errorf("want u-google-1, got %q", user.ID)
	}

	if _, err := svc.CurrentUser(context.Background(), "does-not-exist"); err == nil {
		t.Error("want error for unknown session")
	}
}
