package services

import (
	"context"
	"time"

	"fiber_test/internal/application/ports"
	"fiber_test/internal/domain"
)

// AuthService owns Mixtape sign-in: it turns a Google authorization code into
// a local account plus a session, validates sessions, and signs people out.
type AuthService struct {
	users      ports.UserRepository
	sessions   ports.SessionRepository
	identities ports.GoogleIdentityProvider
	now        func() time.Time
	sessionTTL time.Duration
}

func NewAuthService(
	users ports.UserRepository,
	sessions ports.SessionRepository,
	identities ports.GoogleIdentityProvider,
	now func() time.Time,
	sessionTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:      users,
		sessions:   sessions,
		identities: identities,
		now:        now,
		sessionTTL: sessionTTL,
	}
}

// SignInWithGoogle exchanges the authorization code, creates the account on
// first sign-in, and issues a session.
func (s *AuthService) SignInWithGoogle(ctx context.Context, code, redirectURI string) (domain.Session, error) {
	profile, err := s.identities.ExchangeCode(ctx, code, redirectURI)
	if err != nil {
		return domain.Session{}, err
	}

	user, err := s.users.UpsertByGoogleID(ctx, profile)
	if err != nil {
		return domain.Session{}, err
	}

	return s.sessions.Create(ctx, user.ID, s.sessionTTL)
}

// ValidateSession returns the session when it exists and has not expired.
func (s *AuthService) ValidateSession(ctx context.Context, id string) (domain.Session, error) {
	session, err := s.sessions.Get(ctx, id)
	if err != nil {
		return domain.Session{}, err
	}
	if !session.ExpiresAt.After(s.now()) {
		return domain.Session{}, domain.ErrSessionExpired
	}
	return session, nil
}

// SignOut revokes the session.
func (s *AuthService) SignOut(ctx context.Context, id string) error {
	return s.sessions.Delete(ctx, id)
}

// CurrentUser resolves a session to the account that owns it.
func (s *AuthService) CurrentUser(ctx context.Context, sessionID string) (domain.User, error) {
	session, err := s.ValidateSession(ctx, sessionID)
	if err != nil {
		return domain.User{}, err
	}
	return s.users.ByID(ctx, session.UserID)
}

// SessionTTL is the configured lifetime for new sessions.
func (s *AuthService) SessionTTL() time.Duration {
	return s.sessionTTL
}

// AuthURL returns the Google authorization URL for the given anti-CSRF state.
func (s *AuthService) AuthURL(state string) string {
	return s.identities.AuthURL(state)
}

// SignOutURL returns the Google logout URL used after a local sign-out.
func (s *AuthService) SignOutURL(redirectURI string) string {
	return s.identities.SignOutURL(redirectURI)
}
