# Discord Music Bot - Architecture & Implementation Plan

## Overview

A Discord music bot with a web interface that allows users to:
- Play music from Spotify, YouTube, and SoundCloud links
- Connect their Spotify account to play personal playlists
- Control the bot via Discord commands or web frontend
- Configure bot behavior and channel connections through the frontend

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              DRIVING ADAPTERS (Input)                            │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌──────────────────────┐              ┌──────────────────────┐                │
│  │   Discord Bot        │              │   Web Frontend       │                │
│  │   (Commands)         │              │   (Svelte)           │                │
│  │                      │              │                      │                │
│  │  /play <url>         │              │  - Play/Pause/Skip   │                │
│  │  /skip               │              │  - Queue management  │                │
│  │  /pause              │              │  - Volume control    │                │
│  │  /queue              │              │  - Spotify connect   │                │
│  │  /connect <channel>  │              │  - Channel selector  │                │
│  │  /settings           │              │  - Settings panel    │                │
│  └──────────┬───────────┘              └──────────┬───────────┘                │
│             │                                      │                            │
└─────────────┼──────────────────────────────────────┼────────────────────────────┘
              │                                      │
              ▼                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         APPLICATION LAYER (Ports)                                │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                         Use Cases / Services                            │    │
│  ├─────────────────────────────────────────────────────────────────────────┤    │
│  │  • MusicPlaybackService    - Play, pause, skip, seek                   │    │
│  │  • QueueService            - Add, remove, reorder queue                │    │
│  │  • TrackResolverService    - Resolve URLs to playable tracks           │    │
│  │  • SpotifyIntegrationService - OAuth, playlist import                  │    │
│  │  • GuildConfigService      - Per-guild settings & channel binding      │    │
│  │  • UserPreferencesService  - Per-user settings                         │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
              │                                      │
              ▼                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              DOMAIN LAYER (Core)                                 │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │    Track     │  │   Playlist   │  │  GuildConfig │  │  UserPrefs   │       │
│  ├──────────────┤  ├──────────────┤  ├──────────────┤  ├──────────────┤       │
│  │ ID           │  │ ID           │  │ GuildID      │  │ UserID       │       │
│  │ Title        │  │ Name         │  │ TextChannel  │  │ SpotifyToken │       │
│  │ Artist       │  │ Tracks[]     │  │ VoiceChannel │  │ PreferredSrc │       │
│  │ Duration     │  │ OwnerID      │  │ Volume       │  │ Locale       │       │
│  │ Source       │  │ IsPublic     │  │ AutoPlay     │  │ Notifications│       │
│  │ URL          │  │ Source       │  │ DJRole       │  │              │       │
│  │ AudioURL     │  │              │  │              │  │              │       │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘       │
│                                                                                  │
│  ┌──────────────┐  ┌──────────────┐                                            │
│  │  Queue       │  │  AudioPlayer │                                            │
│  ├──────────────┤  ├──────────────┤                                            │
│  │ GuildID      │  │ State        │                                            │
│  │ Tracks[]     │  │ CurrentTrack │                                            │
│  │ Position     │  │ Volume       │                                            │
│  │ LoopMode     │  │ IsPaused     │                                            │
│  └──────────────┘  └──────────────┘                                            │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
              │                                      │
              ▼                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           DRIVEN ADAPTERS (Output)                               │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Discord    │  │   Spotify    │  │   YouTube    │  │  SoundCloud  │       │
│  │   Gateway    │  │   Web API    │  │   Scraper    │  │   Scraper    │       │
│  │              │  │              │  │              │  │              │       │
│  │  - Gateway   │  │  - OAuth2    │  │  - yt-dlp    │  │  - yt-dlp    │       │
│  │  - REST API  │  │  - Web API   │  │              │  │              │       │
│  │  - Voice     │  │              │  │              │  │              │       │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘       │
│                                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                         │
│  │  PostgreSQL  │  │    Redis     │  │  File Store  │                         │
│  │              │  │              │  │              │                         │
│  │  - Guilds    │  │  - Sessions  │  │  - Audio     │                         │
│  │  - Users     │  │  - Queue     │  │    cache     │                         │
│  │  - Configs   │  │  - Cache     │  │              │                         │
│  └──────────────┘  └──────────────┘  └──────────────┘                         │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

## Hexagonal Architecture Breakdown

### Domain Layer (Core)
Pure business logic with no external dependencies.

**Entities:**
- `Track` - Represents a playable track (source-agnostic)
- `Playlist` - Collection of tracks
- `Queue` - Per-guild playback queue
- `GuildConfig` - Guild-specific settings
- `UserPreferences` - User-specific settings
- `AudioPlayer` - Playback state machine

**Value Objects:**
- `TrackSource` (enum: Spotify, YouTube, SoundCloud)
- `LoopMode` (enum: Off, Single, All)
- `PlaybackState` (enum: Playing, Paused, Stopped)

### Application Layer (Ports)
Defines interfaces (ports) and implements use cases.

**Driving Ports (Inbound):**
```go
type MusicPlaybackPort interface {
    Play(ctx context.Context, guildID string, query string) error
    Pause(ctx context.Context, guildID string) error
    Resume(ctx context.Context, guildID string) error
    Skip(ctx context.Context, guildID string) error
    Seek(ctx context.Context, guildID string, position time.Duration) error
    SetVolume(ctx context.Context, guildID string, volume int) error
}

type QueuePort interface {
    Add(ctx context.Context, guildID string, trackID string) error
    Remove(ctx context.Context, guildID string, index int) error
    Reorder(ctx context.Context, guildID string, from, to int) error
    Clear(ctx context.Context, guildID string) error
    GetQueue(ctx context.Context, guildID string) (*Queue, error)
}

type TrackResolverPort interface {
    Resolve(ctx context.Context, url string) (*Track, error)
    Search(ctx context.Context, query string, source TrackSource) ([]*Track, error)
}

type SpotifyIntegrationPort interface {
    GetAuthURL(userID string) (string, error)
    HandleCallback(ctx context.Context, code, state string) error
    GetUserPlaylists(ctx context.Context, userID string) ([]*Playlist, error)
    PlayPlaylist(ctx context.Context, userID, playlistID string) error
}

type GuildConfigPort interface {
    SetVoiceChannel(ctx context.Context, guildID, channelID string) error
    SetTextChannel(ctx context.Context, guildID, channelID string) error
    SetVolume(ctx context.Context, guildID string, volume int) error
    GetConfig(ctx context.Context, guildID string) (*GuildConfig, error)
}
```

**Driven Ports (Outbound):**
```go
type DiscordRepository interface {
    SendEmbed(ctx context.Context, channelID string, embed *Embed) error
    JoinVoiceChannel(ctx context.Context, guildID, channelID string) error
    LeaveVoiceChannel(ctx context.Context, guildID string) error
    PlayAudio(ctx context.Context, guildID string, audioSource io.Reader) error
    StopAudio(ctx context.Context, guildID string) error
}

type SpotifyRepository interface {
    GetTrack(ctx context.Context, trackID string) (*Track, error)
    GetPlaylist(ctx context.Context, playlistID string) (*Playlist, error)
    GetUserPlaylists(ctx context.Context, accessToken string) ([]*Playlist, error)
    ExchangeCode(ctx context.Context, code string) (*SpotifyTokens, error)
}

type YouTubeRepository interface {
    Search(ctx context.Context, query string) ([]*Track, error)
    GetStream(ctx context.Context, videoID string) (io.ReadCloser, error)
}

type SoundCloudRepository interface {
    Resolve(ctx context.Context, url string) (*Track, error)
    GetStream(ctx context.Context, trackID string) (io.ReadCloser, error)
}

type ConfigRepository interface {
    SaveGuildConfig(ctx context.Context, config *GuildConfig) error
    GetGuildConfig(ctx context.Context, guildID string) (*GuildConfig, error)
    SaveUserPreferences(ctx context.Context, prefs *UserPreferences) error
    GetUserPreferences(ctx context.Context, userID string) (*UserPreferences, error)
}

type CacheRepository interface {
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Get(ctx context.Context, key string, dest interface{}) error
    Delete(ctx context.Context, key string) error
}
```

### Driving Adapters (Input)
**Discord Bot Adapter:**
- Translates Discord slash commands into use case calls
- Handles Discord events (voice state updates, message reactions)
- Sends responses back to Discord

**HTTP API Adapter (Fiber):**
- REST endpoints for frontend communication
- WebSocket endpoint for real-time updates
- OAuth callback handler for Spotify

**Web Frontend (Svelte):**
- Consumes HTTP API
- Real-time updates via WebSocket
- Spotify OAuth flow

### Driven Adapters (Output)
**Discord Gateway Adapter:**
- Implements `DiscordRepository`
- Uses DisGo library

**Spotify API Adapter:**
- Implements `SpotifyRepository`
- Uses Spotify Web API

**YouTube Scraper Adapter:**
- Implements `YouTubeRepository`
- Uses yt-dlp for extraction

**SoundCloud Scraper Adapter:**
- Implements `SoundCloudRepository`
- Uses yt-dlp for extraction

**PostgreSQL Adapter:**
- Implements `ConfigRepository`
- Persists guild configs and user preferences

**Redis Adapter:**
- Implements `CacheRepository`
- Caches queue state and sessions

## Tech Stack

### Backend (Go)
- **Framework:** Fiber v3
- **Discord:** DisGo
- **Database:** PostgreSQL + pgx
- **Cache:** Redis
- **Audio Processing:** yt-dlp (external binary)
- **Spotify:** spotify-web-api (or custom HTTP client)

### Frontend (Svelte)
- **Framework:** Svelte 5 (or SvelteKit if SSR needed)
- **State:** Svelte stores / runes
- **HTTP Client:** Native fetch
- **Real-time:** WebSocket
- **Styling:** TailwindCSS or similar

### Infrastructure
- **Database:** PostgreSQL 15+
- **Cache:** Redis 7+
- **Deployment:** Docker + Docker Compose

## Feature Breakdown

### Phase 1: Core Infrastructure
- [ ] Hexagonal architecture skeleton
- [ ] Discord bot connection
- [ ] Basic command handler
- [ ] Database schema & migrations
- [ ] Redis connection

### Phase 2: Music Playback
- [ ] Track resolver (URL → Track)
- [ ] YouTube support (yt-dlp)
- [ ] Audio streaming to Discord
- [ ] Queue management
- [ ] Basic commands: /play, /skip, /pause, /queue

### Phase 3: Additional Sources
- [ ] SoundCloud support
- [ ] Spotify track/playlist support (metadata only)
- [ ] Search functionality

### Phase 4: Web Frontend
- [ ] Fiber static file serving
- [ ] REST API endpoints
- [ ] WebSocket for real-time updates
- [ ] Svelte app scaffold
- [ ] Queue display & control
- [ ] Playback controls

### Phase 5: Spotify Integration
- [ ] Spotify OAuth flow
- [ ] User account linking
- [ ] Playlist import
- [ ] Play user playlists

### Phase 6: Configuration
- [ ] Guild config commands
- [ ] Frontend config panel
- [ ] Channel binding
- [ ] Volume control
- [ ] Permissions system

### Phase 7: Advanced Features
- [ ] Audio caching
- [ ] Crossfade
- [ ] Lyrics display
- [ ] Vote skip
- [ ] DJ mode

## Database Schema

```sql
-- Guild configurations
CREATE TABLE guild_configs (
    guild_id BIGINT PRIMARY KEY,
    text_channel_id BIGINT,
    voice_channel_id BIGINT,
    default_volume INT DEFAULT 70,
    auto_play BOOLEAN DEFAULT false,
    dj_role_id BIGINT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- User preferences
CREATE TABLE user_preferences (
    user_id BIGINT PRIMARY KEY,
    spotify_access_token TEXT,
    spotify_refresh_token TEXT,
    spotify_token_expiry TIMESTAMP,
    preferred_source VARCHAR(20) DEFAULT 'youtube',
    locale VARCHAR(10) DEFAULT 'en',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Track cache (optional, for faster lookups)
CREATE TABLE track_cache (
    id SERIAL PRIMARY KEY,
    source VARCHAR(20) NOT NULL,
    source_id VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    artist VARCHAR(500),
    duration_ms INT,
    thumbnail_url TEXT,
    cached_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(source, source_id)
);

-- Playlists (user-created or imported)
CREATE TABLE playlists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    owner_id BIGINT NOT NULL,
    source VARCHAR(20),
    source_id VARCHAR(255),
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE playlist_tracks (
    playlist_id INT REFERENCES playlists(id) ON DELETE CASCADE,
    track_id INT REFERENCES track_cache(id) ON DELETE CASCADE,
    position INT NOT NULL,
    PRIMARY KEY (playlist_id, track_id)
);
```

## API Endpoints

### REST API
```
GET    /api/guilds/:guildId/queue          - Get current queue
POST   /api/guilds/:guildId/queue          - Add track to queue
DELETE /api/guilds/:guildId/queue/:index   - Remove from queue
PUT    /api/guilds/:guildId/queue/reorder  - Reorder queue

POST   /api/guilds/:guildId/playback/play    - Play/resume
POST   /api/guilds/:guildId/playback/pause   - Pause
POST   /api/guilds/:guildId/playback/skip    - Skip
POST   /api/guilds/:guildId/playback/seek    - Seek to position
PUT    /api/guilds/:guildId/playback/volume  - Set volume

GET    /api/guilds/:guildId/config         - Get guild config
PUT    /api/guilds/:guildId/config         - Update guild config

GET    /api/users/:userId/preferences      - Get user prefs
PUT    /api/users/:userId/preferences      - Update user prefs

GET    /api/auth/spotify/url               - Get Spotify auth URL
GET    /api/auth/spotify/callback          - Spotify OAuth callback
DELETE /api/auth/spotify                   - Disconnect Spotify

POST   /api/search                         - Search tracks
```

### WebSocket
```
WS /ws/guilds/:guildId

Events (server → client):
- queue_updated
- playback_state_changed
- track_changed
- volume_changed

Events (client → server):
- subscribe
- unsubscribe
```

## Discord Commands

```
/play <url|query>     - Play track or search
/pause                - Pause playback
/resume               - Resume playback
/skip                 - Skip current track
/stop                 - Stop and clear queue
/queue                - Show current queue
/clear                - Clear queue
/shuffle              - Shuffle queue
/loop [mode]          - Set loop mode (off/single/all)
/volume <0-100>       - Set volume
/nowplaying           - Show current track
/search <query>       - Search for tracks

/connect <channel>    - Connect bot to voice channel
/disconnect           - Disconnect bot from voice
/settings             - Show/update guild settings
/spotify connect      - Link Spotify account
/spotify disconnect   - Unlink Spotify account
/spotify playlist     - Play Spotify playlist
```

## Project Structure

```
fiber_test/
├── main.go
├── go.mod
├── go.sum
│
├── internal/
│   ├── domain/              # Domain layer (entities, value objects)
│   │   ├── track.go
│   │   ├── playlist.go
│   │   ├── queue.go
│   │   ├── guild_config.go
│   │   └── user_preferences.go
│   │
│   ├── application/         # Application layer (use cases, ports)
│   │   ├── ports/
│   │   │   ├── inbound.go   # Driving ports
│   │   │   └── outbound.go  # Driven ports
│   │   └── services/
│   │       ├── music_playback.go
│   │       ├── queue.go
│   │       ├── track_resolver.go
│   │       ├── spotify_integration.go
│   │       └── guild_config.go
│   │
│   ├── adapters/            # Adapters layer
│   │   ├── inbound/
│   │   │   ├── discord/     # Discord bot adapter
│   │   │   │   ├── bot.go
│   │   │   │   ├── commands.go
│   │   │   │   └── events.go
│   │   │   └── http/        # HTTP API adapter
│   │   │       ├── router.go
│   │   │       ├── handlers/
│   │   │       └── middleware/
│   │   │
│   │   └── outbound/
│   │       ├── discord/     # Discord repository
│   │       ├── spotify/     # Spotify API
│   │       ├── youtube/     # YouTube scraper
│   │       ├── soundcloud/  # SoundCloud scraper
│   │       ├── postgres/    # Database
│   │       └── redis/       # Cache
│   │
│   └── infrastructure/      # Cross-cutting concerns
│       ├── config/
│       ├── logging/
│       └── errors/
│
├── template/                # Frontend (Svelte)
│   ├── dist/                # Built Svelte app
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   ├── stores/
│   │   │   └── api/
│   │   ├── routes/
│   │   └── App.svelte
│   ├── package.json
│   └── svelte.config.js
│
├── migrations/              # Database migrations
│   └── *.sql
│
├── docs/
│   └── plan.md              # This file
│
└── docker-compose.yml       # Local dev environment
```

## Development Workflow

1. **Start infrastructure:** `docker-compose up -d postgres redis`
2. **Run migrations:** `make migrate-up`
3. **Start backend:** `go run main.go`
4. **Start frontend dev:** `cd template && npm run dev`
5. **Build frontend:** `cd template && npm run build`

## Environment Variables

```env
# Discord
DISCORD_BOT_TOKEN=
DISCORD_APP_ID=

# Spotify
SPOTIFY_CLIENT_ID=
SPOTIFY_CLIENT_SECRET=
SPOTIFY_REDIRECT_URI=http://localhost:3000/api/auth/spotify/callback

# Database
DATABASE_URL=postgres://user:pass@localhost:5432/discord_bot

# Redis
REDIS_URL=redis://localhost:6379

# Server
PORT=3000
ENV=development
```

## Next Steps

1. Set up project structure with hexagonal architecture
2. Implement Discord bot connection
3. Create database schema and migrations
4. Implement basic /play command with YouTube support
5. Build queue management
6. Create web frontend scaffold
7. Implement REST API
8. Add Spotify OAuth
9. Connect frontend to backend
10. Add remaining features incrementally
