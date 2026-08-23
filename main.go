package main

import (
	"context"
	"os/signal"
	"syscall"

	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/adapters/inbound/discord/commands"
	"fiber_test/internal/adapters/inbound/http"
	"fiber_test/internal/adapters/outbound/postgres"
	"fiber_test/internal/adapters/outbound/redis"
	"fiber_test/internal/adapters/outbound/soundcloud"
	"fiber_test/internal/adapters/outbound/spotify"
	"fiber_test/internal/adapters/outbound/youtube"
	"fiber_test/internal/application/services"
	"fiber_test/internal/infrastructure/config"
	"fiber_test/internal/infrastructure/logging"
	"fiber_test/template"
)

func main() {

	cfg := config.Load()
	logger := logging.NewLogger(cfg.Env)

	db, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect to database", "error", err)
	}
	defer db.Close()

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
	youtubeClient := youtube.New()
	soundcloudClient := soundcloud.New()

	trackResolver := services.NewTrackResolverService(youtubeClient, spotifyClient, soundcloudClient)
	queueService := services.NewQueueService(cache)
	musicPlayback := services.NewMusicPlaybackService(queueService, trackResolver, youtubeClient)
	guildConfigService := services.NewGuildConfigService(db)
	spotifyIntegration := services.NewSpotifyIntegrationService(spotifyClient, db)

	httpRouter := http.NewRouter(
		cfg,
		logger,
		musicPlayback,
		queueService,
		guildConfigService,
		spotifyIntegration,
		template.Dist(),
	)

	discordBot := discord.NewBot(
		cfg.DiscordBotToken,
		logger,
		musicPlayback,
		queueService,
		guildConfigService,
		spotifyIntegration,
	)

	discordBot.RegisterCommands(
		commands.NewPlayCommand(musicPlayback),
		commands.NewSkipCommand(musicPlayback),
		commands.NewPauseCommand(musicPlayback),
		commands.NewQueueCommand(musicPlayback, queueService),
		commands.NewVolumeCommand(musicPlayback),
		commands.NewNowPlayingCommand(musicPlayback),
		commands.NewConnectCommand(guildConfigService),
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
