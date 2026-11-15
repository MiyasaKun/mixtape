package config

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v3/log"
	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	Env                 string
	GuildID             string
	DiscordBotToken     string
	DiscordAppID        string
	SpotifyClientID     string
	SpotifyClientSecret string
	SpotifyRedirectURI  string
	GoogleClientID      string
	GoogleClientSecret  string
	GoogleRedirectURI   string
	AuthEncryptionKey   string
	SessionTTL          time.Duration
	DatabaseURL         string
	RedisURL            string
	OTelEndpoint        string
	OTelServiceName     string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading environment")
	}

	return &Config{
		Port:                getEnv("PORT", ":3000"),
		Env:                 getEnv("ENV", "development"),
		DiscordBotToken:     getEnv("DISCORD_BOT_TOKEN", ""),
		DiscordAppID:        getEnv("DISCORD_APP_ID", ""),
		GuildID:             getEnv("GUILD_ID", ""),
		SpotifyClientID:     getEnv("SPOTIFY_CLIENT_ID", ""),
		SpotifyClientSecret: getEnv("SPOTIFY_CLIENT_SECRET", ""),
		SpotifyRedirectURI:  getEnv("SPOTIFY_REDIRECT_URI", "http://localhost:3000/api/auth/spotify/callback"),
		GoogleClientID:      getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:  getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURI:   getEnv("GOOGLE_REDIRECT_URI", "http://localhost:3000/api/auth/google/callback"),
		AuthEncryptionKey:   getEnv("AUTH_ENCRYPTION_KEY", ""),
		SessionTTL:          getEnvDuration("SESSION_TTL", 30*24*time.Hour),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/discord_bot"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		OTelEndpoint:        os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OTelServiceName:     getEnv("OTEL_SERVICE_NAME", "mixtape"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Warn("invalid duration, using default", "key", key, "value", value)
		return defaultValue
	}
	return parsed
}
