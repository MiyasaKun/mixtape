package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fiber_test/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SpotifyRepository is the GORM-backed ports.SpotifyConnectionStore.
type SpotifyRepository struct {
	db *gorm.DB
}

func NewSpotifyRepository(db *Postgres) *SpotifyRepository {
	return &SpotifyRepository{db: db.db}
}

// Save upserts the Spotify connection for a user.
func (r *SpotifyRepository) Save(ctx context.Context, c domain.SpotifyConnection) error {
	now := time.Now().UTC()
	record := spotifyConnectionModel{
		UserID:             c.UserID,
		SpotifyUserID:      c.SpotifyUserID,
		SpotifyDisplayName: c.SpotifyDisplayName,
		AccessToken:        c.AccessToken,
		RefreshTokenCipher: c.RefreshTokenCipher,
		TokenExpiry:        c.TokenExpiry,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"spotify_user_id":      c.SpotifyUserID,
			"spotify_display_name": c.SpotifyDisplayName,
			"access_token":         c.AccessToken,
			"refresh_token_cipher": c.RefreshTokenCipher,
			"token_expiry":         c.TokenExpiry,
			"updated_at":           now,
		}),
	}).Create(&record).Error
	if err != nil {
		return fmt.Errorf("save spotify connection: %w", err)
	}
	return nil
}

// ByUserID returns the user's Spotify connection.
func (r *SpotifyRepository) ByUserID(ctx context.Context, userID string) (domain.SpotifyConnection, error) {
	var record spotifyConnectionModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.SpotifyConnection{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.SpotifyConnection{}, fmt.Errorf("get spotify connection: %w", err)
	}
	return domain.SpotifyConnection{
		UserID:             record.UserID,
		SpotifyUserID:      record.SpotifyUserID,
		SpotifyDisplayName: record.SpotifyDisplayName,
		AccessToken:        record.AccessToken,
		RefreshTokenCipher: record.RefreshTokenCipher,
		TokenExpiry:        record.TokenExpiry,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}, nil
}

// Delete removes the user's Spotify connection.
func (r *SpotifyRepository) Delete(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&spotifyConnectionModel{}).Error; err != nil {
		return fmt.Errorf("delete spotify connection: %w", err)
	}
	return nil
}
