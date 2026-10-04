package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"fiber_test/internal/domain"

	"gorm.io/gorm"
)

// SessionsRepository is the GORM-backed ports.SessionRepository.
type SessionsRepository struct {
	db *gorm.DB
}

func NewSessionsRepository(db *Postgres) *SessionsRepository {
	return &SessionsRepository{db: db.db}
}

// Create issues a new session with an opaque random ID.
func (r *SessionsRepository) Create(ctx context.Context, userID string, ttl time.Duration) (domain.Session, error) {
	id, err := newSessionID()
	if err != nil {
		return domain.Session{}, err
	}

	record := sessionModel{
		ID:        id,
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return domain.Session{}, fmt.Errorf("create session: %w", err)
	}

	return domain.Session{
		ID:        record.ID,
		UserID:    record.UserID,
		ExpiresAt: record.ExpiresAt,
		CreatedAt: record.CreatedAt,
	}, nil
}

// Get returns a session by ID.
func (r *SessionsRepository) Get(ctx context.Context, id string) (domain.Session, error) {
	var record sessionModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("get session: %w", err)
	}
	return domain.Session{
		ID:        record.ID,
		UserID:    record.UserID,
		ExpiresAt: record.ExpiresAt,
		CreatedAt: record.CreatedAt,
	}, nil
}

// Delete revokes one session.
func (r *SessionsRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&sessionModel{}).Error; err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpired removes sessions past their expiry.
func (r *SessionsRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Where("expires_at <= ?", time.Now()).Delete(&sessionModel{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func newSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
