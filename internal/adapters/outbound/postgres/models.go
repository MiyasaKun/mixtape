package postgres

import "time"

// These structs are GORM persistence models. They are mapped to the pure
// domain types and never leave this package.

type userModel struct {
	ID       string `gorm:"column:id;type:uuid;primaryKey"`
	GoogleID string `gorm:"column:google_id;uniqueIndex;not null"`
	Email    string `gorm:"column:email;not null"`
	Name     string `gorm:"column:name;not null;default:''"`
	Picture  string `gorm:"column:picture;not null;default:''"`
	Locale   string `gorm:"column:locale;type:varchar(10);not null;default:'en'"`
	// SpotifyAutoplay is an account-level web preference: when true, “Open in
	// Spotify” links request autoplay.
	SpotifyAutoplay bool      `gorm:"column:spotify_autoplay;not null;default:false"`
	CreatedAt       time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`

	Connection *spotifyConnectionModel `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (userModel) TableName() string { return "users" }

type sessionModel struct {
	ID        string    `gorm:"column:id;primaryKey"`
	UserID    string    `gorm:"column:user_id;type:uuid;not null;index:idx_sessions_user_id"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null;index:idx_sessions_expires_at"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (sessionModel) TableName() string { return "sessions" }

type spotifyConnectionModel struct {
	UserID             string    `gorm:"column:user_id;type:uuid;primaryKey"`
	SpotifyUserID      string    `gorm:"column:spotify_user_id;not null"`
	SpotifyDisplayName string    `gorm:"column:spotify_display_name;not null;default:''"`
	AccessToken        string    `gorm:"column:access_token;not null"`
	RefreshTokenCipher string    `gorm:"column:refresh_token_cipher;not null"`
	TokenExpiry        time.Time `gorm:"column:token_expiry;not null"`
	CreatedAt          time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (spotifyConnectionModel) TableName() string { return "spotify_connections" }
