# Discord Music Bot

A Discord music bot with web dashboard, built with Go (Fiber) and SvelteKit.

## Features

- Music playback from YouTube, Spotify, and SoundCloud
- Discord slash commands for controlling playback
- Web dashboard for managing the bot
- Queue management
- Volume control
- Guild-specific configuration

## Tech Stack

- **Backend**: Go with Fiber framework
- **Frontend**: SvelteKit with TailwindCSS
- **Database**: PostgreSQL
- **Cache**: Redis
- **Discord**: DisGo

## Requirements

- Go 1.26+
- Bun (for frontend)
- PostgreSQL
- Redis
- Discord Bot Token

## Setup

1. Clone the repository:
```sh
git clone <repo-url>
cd fiber_test
```

2. Copy `.env.example` to `.env` and configure:
```sh
cp .env.example .env
```

3. Install dependencies:
```sh
# Backend
go mod download

# Frontend
cd template && bun install
```

4. Run with Docker:
```sh
docker compose up --build
```

Or manually:
```sh
# Start frontend dev server
cd template && bun run dev

# Start backend
go run main.go
```

## Commands

| Command | Description |
|---------|-------------|
| `/play <query>` | Play a song or add to queue |
| `/skip` | Skip current song |
| `/pause` | Pause/resume playback |
| `/queue` | View the queue |
| `/volume <level>` | Set volume (0-100) |
| `/nowplaying` | Show current song |
| `/connect` | Connect bot to voice channel |

## Building

```sh
# Build frontend
cd template && bun run build

# Build backend
go build -o bot main.go
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DISCORD_BOT_TOKEN` | Discord bot token |
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `SPOTIFY_CLIENT_ID` | Spotify API client ID |
| `SPOTIFY_CLIENT_SECRET` | Spotify API client secret |
| `SPOTIFY_REDIRECT_URI` | Spotify OAuth redirect URI |
| `PORT` | HTTP server port (default: 3000) |
