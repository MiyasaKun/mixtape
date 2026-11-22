package ports

// VoiceGuild is a guild where the bot currently has an active voice
// connection. It is what the web UI offers as a playback target.
type VoiceGuild struct {
	ID   string
	Name string
}

// VoiceGuildLister exposes the bot's active voice connections to inbound
// adapters without coupling them to the Discord implementation.
type VoiceGuildLister interface {
	ActiveVoiceGuilds() []VoiceGuild
}
