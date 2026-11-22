package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/adapters/inbound/discord/commands"
	"fiber_test/internal/adapters/inbound/http"
	"fiber_test/internal/adapters/outbound/crypto"
	"fiber_test/internal/adapters/outbound/google"
	"fiber_test/internal/adapters/outbound/postgres"
	"fiber_test/internal/adapters/outbound/redis"
	"fiber_test/internal/adapters/outbound/soundcloud"
	"fiber_test/internal/adapters/outbound/spotify"
	"fiber_test/internal/adapters/outbound/youtube"
	"fiber_test/internal/application/services"
	"fiber_test/internal/infrastructure/config"
	"fiber_test/internal/infrastructure/logging"
	"fiber_test/internal/infrastructure/otel"
	"fiber_test/template"
)

func main() {

	cfg := config.Load()
	logger := logging.NewLogger(cfg.Env)

	// Telemetry is opt-in: with OTEL_EXPORTER_OTLP_ENDPOINT unset this is a
	// disabled setup handing out no-op sinks.
	ctx, stopTelemetry := context.WithCancel(context.Background())
	telemetry, err := otel.Setup(ctx, otel.Config{
		Endpoint:    cfg.OTelEndpoint,
		ServiceName: cfg.OTelServiceName,
		Env:         cfg.Env,
	})
	if err != nil {
		logger.Error("failed to initialize telemetry", "error", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetry.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to flush telemetry", "error", err)
		}
	}()
	defer stopTelemetry()

	// Console stays the primary sink; OTLP logs ride alongside when enabled.
	if telemetry.Enabled {
		logger = logging.NewLogger(cfg.Env, telemetry.LogHandler)
	}

	db, err := postgres.New(cfg.DatabaseURL, logger, cfg.Env)
	if err != nil {
		logger.Fatal("failed to connect to database", "error", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", "error", err)
		}
	}()

	if err := db.Migrate(); err != nil {
		logger.Fatal("failed to migrate database", "error", err)
	}

	cache, err := redis.New(cfg.RedisURL)
	if err != nil {
		logger.Fatal("failed to connect to redis", "error", err)
	}
	defer func(cache *redis.Redis) {
		err := cache.Close()
		if err != nil {

		}
	}(cache)

	spotifyClient := spotify.New(cfg.SpotifyClientID, cfg.SpotifyClientSecret, cfg.SpotifyRedirectURI)
	googleClient := google.New(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURI)
	youtubeClient := youtube.New()
	soundcloudClient := soundcloud.New()
	metrics := telemetry.Metrics()
	telemetryDep := services.Telemetry{Logger: logger, Metrics: metrics}

	encryptor, err := crypto.New(cfg.AuthEncryptionKey)
	if err != nil {
		logger.Fatal("failed to initialize auth encryption", "error", err)
	}

	users := postgres.NewUserRepository(db)
	sessions := postgres.NewSessionsRepository(db)
	spotifyConnections := postgres.NewSpotifyRepository(db)

	authService := services.NewAuthService(users, sessions, googleClient, time.Now, cfg.SessionTTL)

	trackResolver := services.NewTrackResolverService(youtubeClient, spotifyClient, soundcloudClient, telemetryDep)
	queueService := services.NewQueueService(cache, telemetryDep)
	musicPlayback := services.NewMusicPlaybackService(queueService, trackResolver, youtubeClient, telemetryDep)
	guildConfigService := services.NewGuildConfigService(db)
	spotifyIntegration := services.NewSpotifyIntegrationService(spotifyClient, spotifyConnections, encryptor, time.Now)
	webPlayback := services.NewWebPlaybackService(trackResolver, musicPlayback)

	discordBot := discord.NewBot(
		cfg.DiscordBotToken,
		logger,
		metrics,
		musicPlayback,
		queueService,
		guildConfigService,
		spotifyIntegration,
	)

	httpRouter := http.NewRouter(
		cfg,
		logger,
		telemetry.HTTPMetrics(),
		musicPlayback,
		queueService,
		guildConfigService,
		spotifyIntegration,
		webPlayback,
		discordBot,
		authService,
		template.Dist(),
	)

	discordBot.RegisterCommands(
		commands.NewPlayCommand(musicPlayback, trackResolver, discordBot.Selections(), logger),
		commands.NewSkipCommand(musicPlayback),
		commands.NewQueueCommand(musicPlayback, queueService),
		commands.NewNowPlayingCommand(musicPlayback),
		commands.NewConnectCommand(guildConfigService, musicPlayback),
	)

	if err := discordBot.Start(); err != nil {
		logger.Fatal("failed to start discord bot", "error", err)
	}

	defer discordBot.Stop()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		logger.Info("shutting down server...")
		if err := httpRouter.Shutdown(); err != nil {
			logger.Error("error shutting down server", "error", err)
		}
	}()

	logger.Info("starting server", "port", cfg.Port)
	if err := httpRouter.Listen(cfg.Port); err != nil {
		logger.Fatal("server error", "error", err)
	}
}
