package postgres

import (
	"context"
	"fmt"

	"fiber_test/internal/application/ports"
	"fiber_test/internal/infrastructure/logging"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Postgres wraps the GORM database handle and its shared adapter methods.
type Postgres struct {
	db *gorm.DB
}

// New opens a GORM connection over the pgx driver, configures SQL logging, and
// verifies the connection with a ping. Schema creation is a separate call to
// Migrate so callers control when it happens.
func New(databaseURL string, appLogger *logging.Logger, env string) (*Postgres, error) {
	level := logger.Error
	if env == "production" {
		level = logger.Warn
	}

	db, err := gorm.Open(gormpostgres.Open(databaseURL), &gorm.Config{
		Logger: logger.New(slogWriter{logger: appLogger}, logger.Config{
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to access database handle: %w", err)
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Postgres{db: db}, nil
}

// Migrate creates or updates the schema from the GORM models. It is the single
// source of truth for the database schema.
func (p *Postgres) Migrate() error {
	if err := p.db.AutoMigrate(&userModel{}, &spotifyConnectionModel{}, &sessionModel{}); err != nil {
		return fmt.Errorf("auto-migrate schema: %w", err)
	}
	return nil
}

// Close releases the underlying connection pool.
func (p *Postgres) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// DB exposes the GORM handle for repositories in this package.
func (p *Postgres) DB() *gorm.DB {
	return p.db
}

// Compile-time checks that the repositories satisfy their ports.
var (
	_ ports.UserRepository         = (*UserRepository)(nil)
	_ ports.SessionRepository      = (*SessionsRepository)(nil)
	_ ports.SpotifyConnectionStore = (*SpotifyRepository)(nil)
)

// slogWriter adapts GORM's logger to the application's slog logger.
type slogWriter struct {
	logger *logging.Logger
}

func (w slogWriter) Printf(format string, args ...any) {
	w.logger.Error(fmt.Sprintf(format, args...))
}
