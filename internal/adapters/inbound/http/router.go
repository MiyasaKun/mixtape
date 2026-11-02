package http

import (
	"fiber_test/internal/adapters/inbound/http/handlers"
	"fiber_test/internal/adapters/inbound/http/middleware"
	"fiber_test/internal/application/services"
	"fiber_test/internal/infrastructure/config"
	"fiber_test/internal/infrastructure/logging"
	"io"
	"io/fs"

	"github.com/gofiber/contrib/v3/swaggerui"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/static"
)

type Router struct {
	app      *fiber.App
	staticFS fs.FS
}

func NewRouter(
	cfg *config.Config,
	appLogger *logging.Logger,
	httpMetrics middleware.HTTPMetrics,
	musicPlayback *services.MusicPlaybackService,
	queue *services.QueueService,
	guildConfig *services.GuildConfigService,
	spotifyIntegration *services.SpotifyIntegrationService,
	authService *services.AuthService,
	staticFS fs.FS,
) *Router {
	r := &Router{
		staticFS: staticFS,
	}

	app := fiber.New(fiber.Config{
		AppName: "Discord Music Bot",
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"*"},
	}))

	// Correlation first so every later log line and metric can carry the id.
	app.Use(requestid.New())
	app.Use(middleware.ObserveRequests(httpMetrics))
	app.Use(middleware.RequestLog(appLogger))
	app.Use(swaggerui.New(swaggerui.Config{
		BasePath: "/",
		FilePath: "./docs/swagger.json",
		Path:     "docs/swagger",
		Title:    "Discord Music Bot API",
	}))

	api := app.Group("/api")

	api.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	auth := handlers.NewAuthHandler(
		authService,
		spotifyIntegration,
		cfg.GoogleRedirectURI,
		cfg.SpotifyRedirectURI,
		cfg.Env == "production",
	)

	authGroup := api.Group("/auth")
	authGroup.Get("/google/url", auth.GoogleURL)
	authGroup.Get("/google/callback", auth.GoogleCallback)
	authGroup.Get("/spotify/url", middleware.RequireAuth(authService), auth.SpotifyURL)
	authGroup.Get("/spotify/callback", middleware.RequireAuth(authService), auth.SpotifyCallback)
	authGroup.Delete("/spotify", middleware.RequireAuth(authService), auth.SpotifyDisconnect)
	authGroup.Get("/me", middleware.RequireAuth(authService), auth.Me)
	authGroup.Post("/logout", auth.Logout)

	playlists := handlers.NewPlaylistHandler(spotifyIntegration)
	spotifyGroup := api.Group("/spotify", middleware.RequireAuth(authService))
	spotifyGroup.Get("/playlists", playlists.ListPlaylists)
	spotifyGroup.Get("/playlists/:id/tracks", playlists.ListPlaylistTracks)
	spotifyGroup.Get("/liked", playlists.ListLikedTracks)

	app.Use(static.New("/", static.Config{
		FS:              staticFS,
		IndexNames:      []string{"index.html"},
		NotFoundHandler: r.spaFallback,
	}))

	r.app = app
	return r
}

func (r *Router) spaFallback(c fiber.Ctx) error {
	file, err := r.staticFS.Open("index.html")
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Not found")
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(content)
}

func (r *Router) Listen(addr string) error {
	return r.app.Listen(addr)
}

func (r *Router) Shutdown() error {
	return r.app.Shutdown()
}
