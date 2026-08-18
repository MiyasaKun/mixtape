package config

import (
	"os"

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
	DatabaseURL         string
	RedisURL            string
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
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/discord_bot"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
