# AGENTS.md — mixtape (Discord Music Bot)

Go Fiber backend + SvelteKit SPA. Svelte = client logic only. Go = backend logic only.

## Architecture

### Backend (`/`, module `fiber_test`, Go 1.27, Fiber v3)
Hexagonal / clean architecture. Dependency direction: `domain` <- `application` <- `adapters` <- `main.go`. `infrastructure` holds config/logging.

- `main.go` — composition root only. Manual DI: `config.Load` → `logging` → `postgres.New` → `redis.New` → spotify/youtube/soundcloud clients → `TrackResolver/Queue/MusicPlayback/GuildConfig/SpotifyIntegration` services → `http.NewRouter(..., template.Dist())` + `discord.NewBot` + `RegisterCommands(play/skip/queue/nowplaying/connect)`. Graceful shutdown via `signal.NotifyContext`.
- `internal/domain/` — pure types (`track, queue, playlist, guild_config, user_preferences`). Mostly stubs — implement here first, no external imports.
- `internal/application/ports/` — boundaries. `inbound.go: AudioConn`, `outbound.go: FrameProvider, AudioStreamer`. New cross-layer deps must go through ports as interfaces.
- `internal/application/services/` — use cases: `queue.go` (Redis-backed), `music_playback.go`, `track_resolver.go` (youtube/spotify/soundcloud URL detection), `guild_config.go`, `spotify_integration.go`. Services take interfaces, never concrete adapters.
- `internal/adapters/inbound/http/router.go` — Fiber setup: `cors(*)`, request logger, swagger UI at `docs/swagger` (`FilePath ./docs/swagger.json`), `GET /api/health -> {status:ok}`, SPA static serve with `index.html` fallback. Handlers in `handlers/` are stubs — wire new routes under `api := app.Group(/api)`.
- `internal/adapters/inbound/discord/` — disgo bot (`bot.go`), `command.go` interface + converters, `audio.go` voice adapters, `commands/` thin wrappers over services only. No business logic in commands.
- `internal/adapters/outbound/` — `postgres/` (pgxpool wrapper), `redis/` (go-redis wrapper), `youtube/` (only fully implemented: yt-dlp+ffmpeg → PCM → gopus Opus frames), `spotify/`, `soundcloud/`, `discord/` stubs.
- `internal/infrastructure/config/config.go` — `godotenv.Load` + `getEnv` defaults (`:3000`, `development`). `logging/logging.go` — slog+tint dev / JSON prod.
- `template/template.go` — `//go:embed all:build` + `Dist() fs.FS`. Backend serves Svelte output at `/`.
- `migrations/001_initial_schema.sql` — stub (`guilds` table). `docs/` — swagger source.
- `test/unit/...` (`go test ./test/unit/...`), `test/integration/...` (`go test -tags=integration ./test/integration/...`, needs `REDIS_URL`).

### Frontend (`template/`, SvelteKit 2 + Svelte 5 SPA)
Client logic only. No SSR, no `+server.ts` / `+page.server.ts` / `hooks.*`, no direct DB/Redis/Discord access. All data via backend `/api/*` (`src/lib/api/client.js` — currently empty stub, implement fetch wrapper here).

- Build: `adapter-static({pages:'build', assets:'build', fallback:'index.html'})`, `src/routes/+layout.ts: ssr=false`, Tailwind v4 + DaisyUI v5, paraglide i18n (`messages/en.json,de.json`, `project.inlang/settings.json`, generated `src/paraglide/*` — never edit generated).
- `src/routes/`: `+layout.svelte` (navbar + locale dropdown), `+page.svelte` (home), `dashboard/`, `test/`, `showcase/` (491-line design gallery, self-contained demo). `layout.css` holds neumorphic theme + `@utility neu/*`.
- Stubs to implement: `src/lib/api/client.ts`, `src/lib/stores/queue.ts`, `src/lib/stores/playback.ts`, `src/lib/components/Player.svelte,Queue.svelte,Settings.svelte`.
- The Frontend uses Typescript for all of its logic. 
- Build output `build/` is embedded by Go — never import Go code from `template/`, never render HTML from Go except SPA fallback.

## Mandatory Skills

Invoke via the `skill` tool before acting. Do not skip.

- **All changes: `grilling` + `tdd`.**
  - `grilling`: stress-test the plan/decision first. Challenge assumptions, edge cases, alternatives. Required before implementing any feature, fix, or refactor.
  - `tdd`: red-green-refactor. Write failing test first, then minimal implementation, then refactor. Backend: `test/unit/` for services/domain, `test/integration/` (tag `integration`) for Redis/Postgres. Frontend: component/store test or explicit manual verification plan if no runner is configured.
- **Frontend changes (`template/**`): add `frontend-design` + `web-design-guidelines`.**
  - `frontend-design`: distinctive, intentional UI — no generic templated look. Reuse `layout.css` theme vars, DaisyUI tokens, Archivo Black.
  - `web-design-guidelines`: accessibility/UX audit (keyboard, focus, contrast, responsive, toasts/status, light/dark via existing toggle pattern in `showcase/`).
  - Frontend skill set = `grilling` + `tdd` + `frontend-design` + `web-design-guidelines`.

## Rules

1. Separation: Svelte = presentation, stores, API calls. Go = domain, services, adapters, `/api/*`. No `fetch` to third-party music APIs from Svelte; no HTML/CSS in Go except static file serving.
2. Follow existing layering: domain types → ports interfaces → services → adapters. No service importing `adapters/inbound/http` or Discord command importing `outbound/postgres` directly.
3. Extend `docs/swagger.json` when adding/changing `/api/*` routes.
4. Rebuild contract: frontend change requires `cd template && bun run build` so `template/build/` + Go embed stay in sync. Backend entry: `go build -o app` / `go run main.go` (`makefile`: `all: svelte-deps svelte-build go-build`).
5. Verify by execution: `go test ./test/unit/...`, `svelte-check` (`cd template && bun run check`), targeted integration tests when touching queue/cache. Prefer `read`/`edit`/`write` tools over shell for file ops; `shell` only for builds/tests.
6. Keep edits minimal; prefer editing existing files over creating new ones. Do not edit `src/paraglide/*`, `template/build/*`, `.svelte-kit/*`.

## Security — Sensitive Files

NEVER request, read, cat, print, or otherwise inspect `.env` or other sensitive files: `.env*` (except `.env.example`), `*.pem`, `*.key`, `*secret*`, `*token*`, `*credentials*`, `service-account*.json`. Use `.env.example` key names only (`DISCORD_BOT_TOKEN/APP_ID`, `SPOTIFY_CLIENT_ID/SECRET/REDIRECT_URI`, `DATABASE_URL`, `REDIS_URL`, `PORT`, `ENV`). `.env` is gitignored — keep it that way.
