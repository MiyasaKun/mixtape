package discord_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"

	dgo "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func TestSelectionStorePutAssignsUniqueRequestIDs(t *testing.T) {
	store := discord.NewSelectionStore(time.Minute)
	id1 := store.Put("user-1", "guild-1", "", []services.Track{{Title: "a", URL: "u1"}, {Title: "b", URL: "u2"}})
	id2 := store.Put("user-2", "guild-1", "", []services.Track{{Title: "c", URL: "u3"}})
	if id1 == "" || id2 == "" || id1 == id2 {
		t.Fatalf("want distinct non-empty request IDs, got %q and %q", id1, id2)
	}
}

func TestSelectionStorePutStoresCoverURL(t *testing.T) {
	store := discord.NewSelectionStore(time.Minute)
	cover := "https://i.scdn.co/album.jpg"
	id := store.Put("user-1", "guild-1", cover, []services.Track{{Title: "a", URL: "u1"}})

	tracks, gotCover, ok := store.Take(id)
	if !ok || len(tracks) != 1 {
		t.Fatalf("want candidates back, got %v ok=%v", tracks, ok)
	}
	if gotCover != cover {
		t.Errorf("cover = %q, want %q", gotCover, cover)
	}
}

func TestSelectionStoreTakeReturnsAndRemoves(t *testing.T) {
	store := discord.NewSelectionStore(time.Minute)
	candidates := []services.Track{{Title: "a", URL: "u1"}, {Title: "b", URL: "u2"}}
	id := store.Put("user-1", "guild-1", "", candidates)

	got, _, ok := store.Take(id)
	if !ok || len(got) != 2 || got[1].Title != "b" {
		t.Fatalf("want candidates back, got %v ok=%v", got, ok)
	}
	if _, _, ok := store.Take(id); ok {
		t.Fatal("want second Take to miss (single use)")
	}
}

func TestSelectionStoreTakeExpiresAfterTTL(t *testing.T) {
	store := discord.NewSelectionStore(10 * time.Millisecond)
	id := store.Put("user-1", "guild-1", "", []services.Track{{Title: "a", URL: "u1"}})
	time.Sleep(30 * time.Millisecond)

	if _, _, ok := store.Take(id); ok {
		t.Fatal("want expired selection to be dropped")
	}
}

func TestParseSpotifyPickCustomID(t *testing.T) {
	requestID := "req-123"

	t.Run("valid", func(t *testing.T) {
		id := discord.SpotifyPickCustomID("12345", "67890", requestID, 2)
		guild, user, gotReq, index, ok := discord.ParseSpotifyPickCustomID(id)
		if !ok {
			t.Fatalf("want ok for %q", id)
		}
		if guild != "12345" || user != "67890" || gotReq != requestID || index != 2 {
			t.Errorf("got guild=%q user=%q req=%q index=%d", guild, user, gotReq, index)
		}
	})

	t.Run("foreign custom id", func(t *testing.T) {
		if _, _, _, _, ok := discord.ParseSpotifyPickCustomID("other:1:2:3:4"); ok {
			t.Error("want not ok for unrelated custom ID")
		}
		if _, _, _, _, ok := discord.ParseSpotifyPickCustomID("garbage"); ok {
			t.Error("want not ok for malformed custom ID")
		}
	})
}

func TestBuildSpotifyPickButtons(t *testing.T) {
	candidates := []services.Track{
		{Title: "Nightcall (Official Audio)", Artist: "Kavinsky", Duration: 258, ID: "vid-1"},
		{Title: "Nightcall - Live", Artist: "Kavinsky TV", Duration: 292, ID: "vid-2"},
		{Title: "NoArtistTrack", Duration: 61, ID: "vid-3"},
	}
	requestID := "req-1"
	buttons := discord.BuildSpotifyPickButtons("12345", "67890", requestID, candidates)
	if len(buttons) != 3 {
		t.Fatalf("want 3 buttons, got %d", len(buttons))
	}
	for i, b := range buttons {
		button := b.(dgo.ButtonComponent)
		if button.CustomID != discord.SpotifyPickCustomID("12345", "67890", requestID, i) {
			t.Errorf("button %d custom ID = %q, want index %d encoded", i, button.CustomID, i)
		}
		if button.Style != dgo.ButtonStylePrimary {
			t.Errorf("button %d style = %v, want primary", i, button.Style)
		}
		want := fmt.Sprintf("%d", i+1)
		if button.Label != want {
			t.Errorf("button %d label = %q, want %q", i, button.Label, want)
		}
	}
}

func TestSpotifyPickList(t *testing.T) {
	candidates := []services.Track{
		{Title: "Nightcall (Official Audio)", Artist: "Kavinsky", Duration: 258},
		{Title: "Nightcall - Live", Artist: "Kavinsky TV", Duration: 292},
	}
	list := discord.SpotifyPickList(candidates)
	if !strings.Contains(list, "1. Kavinsky – Nightcall (Official Audio) (4:18)") {
		t.Errorf("want numbered first entry, got %q", list)
	}
	if !strings.Contains(list, "2. Kavinsky TV – Nightcall - Live (4:52)") {
		t.Errorf("want numbered second entry, got %q", list)
	}
}

func TestSpotifyPickCustomIDWithinDiscordLimit(t *testing.T) {
	// Discord limits custom IDs to 100 bytes; snowflake IDs are the realistic
	// upper bound for guild/user.
	longReq := "abcdef0123456789"
	id := discord.SpotifyPickCustomID(snowflake.ID(1).String(), snowflake.ID(2).String(), longReq, 4)
	if len(id) > 100 {
		t.Errorf("custom ID too long: %d", len(id))
	}
}
