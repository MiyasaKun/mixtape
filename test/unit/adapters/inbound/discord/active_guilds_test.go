package discord_test

import (
	"iter"
	"testing"

	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/ports"

	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
)

// fakeVoiceConn embeds the interface so only the two methods the guild
// listing reads need any behaviour.
type fakeVoiceConn struct {
	voice.Conn
	guildID   snowflake.ID
	channelID *snowflake.ID
}

func (f fakeVoiceConn) GuildID() snowflake.ID    { return f.guildID }
func (f fakeVoiceConn) ChannelID() *snowflake.ID { return f.channelID }

func connSeq(conns ...voice.Conn) iter.Seq[voice.Conn] {
	return func(yield func(voice.Conn) bool) {
		for _, conn := range conns {
			if !yield(conn) {
				return
			}
		}
	}
}

func guildNames(names map[snowflake.ID]string) func(snowflake.ID) (string, bool) {
	return func(id snowflake.ID) (string, bool) {
		name, ok := names[id]
		return name, ok
	}
}

func TestActiveVoiceGuildsListsOnlyOpenConnections(t *testing.T) {
	openChannel := snowflake.MustParse("100000000000000010")
	neverOpened := snowflake.MustParse("100000000000000020")
	conns := connSeq(
		fakeVoiceConn{guildID: snowflake.MustParse("200000000000000010"), channelID: &openChannel},
		fakeVoiceConn{guildID: neverOpened, channelID: nil},
	)

	guilds := discord.ActiveVoiceGuilds(conns, guildNames(map[snowflake.ID]string{
		snowflake.MustParse("200000000000000010"): "Studio",
	}))

	if len(guilds) != 1 {
		t.Fatalf("want only the connection with an open channel, got %+v", guilds)
	}
	if guilds[0] != (ports.VoiceGuild{ID: "200000000000000010", Name: "Studio"}) {
		t.Errorf("unexpected guild: %+v", guilds[0])
	}
}

func TestActiveVoiceGuildsSortsByNameAndFallsBackToID(t *testing.T) {
	channel := snowflake.MustParse("100000000000000010")
	first := snowflake.MustParse("200000000000000010")
	second := snowflake.MustParse("200000000000000020")
	conns := connSeq(
		fakeVoiceConn{guildID: second, channelID: &channel},
		fakeVoiceConn{guildID: first, channelID: &channel},
	)

	guilds := discord.ActiveVoiceGuilds(conns, guildNames(map[snowflake.ID]string{
		first:  "Zeta",
		second: "Alpha",
	}))

	if len(guilds) != 2 || guilds[0].Name != "Alpha" || guilds[1].Name != "Zeta" {
		t.Fatalf("want name-sorted guilds, got %+v", guilds)
	}

	unnamed := discord.ActiveVoiceGuilds(conns, func(snowflake.ID) (string, bool) { return "", false })
	if len(unnamed) != 2 || unnamed[0].ID != first.String() || unnamed[1].ID != second.String() {
		t.Fatalf("want id-sorted guilds when names are unknown, got %+v", unnamed)
	}
}
