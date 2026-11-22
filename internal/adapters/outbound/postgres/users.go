package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fiber_test/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRepository is the GORM-backed ports.UserRepository.
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *Postgres) *UserRepository {
	return &UserRepository{db: db.db}
}

// UpsertByGoogleID creates the account on first sign-in and refreshes the
// profile fields on later sign-ins.
func (r *UserRepository) UpsertByGoogleID(ctx context.Context, profile domain.GoogleProfile) (domain.User, error) {
	now := time.Now().UTC()
	record := userModel{
		ID:        uuid.NewString(),
		GoogleID:  profile.GoogleID,
		Email:     profile.Email,
		Name:      profile.Name,
		Picture:   profile.Picture,
		Locale:    "en",
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "google_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"email":      profile.Email,
			"name":       profile.Name,
			"picture":    profile.Picture,
			"updated_at": now,
		}),
	}).Create(&record).Error
	if err != nil {
		return domain.User{}, fmt.Errorf("upsert user: %w", err)
	}

	// On conflict the INSERT above does not populate the stored row, so read
	// it back to return the existing account's ID and timestamps.
	var stored userModel
	if err := r.db.WithContext(ctx).Where("google_id = ?", profile.GoogleID).First(&stored).Error; err != nil {
		return domain.User{}, fmt.Errorf("load upserted user: %w", err)
	}
	return toDomainUser(stored), nil
}

// ByID returns the account plus its Spotify connection state.
func (r *UserRepository) ByID(ctx context.Context, id string) (domain.User, error) {
	var record userModel
	err := r.db.WithContext(ctx).Preload("Connection").Where("id = ?", id).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	return toDomainUser(record), nil
}

// UpdateSpotifyAutoplay persists the account-level autoplay preference.
func (r *UserRepository) UpdateSpotifyAutoplay(ctx context.Context, id string, enabled bool) error {
	result := r.db.WithContext(ctx).
		Model(&userModel{}).
		Where("id = ?", id).
		Update("spotify_autoplay", enabled)
	if result.Error != nil {
		return fmt.Errorf("update spotify autoplay: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func toDomainUser(m userModel) domain.User {
	return domain.User{
		ID:               m.ID,
		GoogleID:         m.GoogleID,
		Email:            m.Email,
		Name:             m.Name,
		Picture:          m.Picture,
		Locale:           m.Locale,
		SpotifyConnected: m.Connection != nil,
		SpotifyAutoplay:  m.SpotifyAutoplay,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
