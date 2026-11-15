package discord_test

import (
	"strings"
	"testing"
	"time"

	"fiber_test/internal/adapters/inbound/discord"
	"fiber_test/internal/application/services"
)

// playingStatus builds a status snapshot for a track playing at elapsed.
func playingStatus(elapsed time.Duration) *services.PlaybackStatus {
	return &services.PlaybackStatus{
		Track: &services.Track{
			Title:        "Nightcall",
			Artist:       "Kavinsky",
			Duration:     258,
			Source:       "youtube",
			ThumbnailURL: "https://i.ytimg.com/thumb.jpg",
		},
		Elapsed: elapsed,
	}
}

func TestBuildNowPlayingPanelEmbed(t *testing.T) {
	embeds := discord.BuildNowPlayingPanel(playingStatus(35*time.Second), time.Now())
	if len(embeds) != 1 {
		t.Fatalf("want 1 embed, got %d", len(embeds))
	}

	embed := embeds[0]
	if embed.Title != "🎵 Now Playing" {
		t.Errorf("title = %q", embed.Title)
	}
	if !strings.Contains(embed.Description, "**Nightcall**") {
		t.Errorf("description %q missing bold title", embed.Description)
	}
	if !strings.Contains(embed.Description, "by Kavinsky") {
		t.Errorf("description %q missing artist", embed.Description)
	}
	if embed.Color != 0x5865F2 {
		t.Errorf("color = %d, want 0x5865F2", embed.Color)
	}
	if embed.Footer == nil || embed.Footer.Text != "YouTube • Mixtape" {
		t.Errorf("footer = %+v, want YouTube • Mixtape", embed.Footer)
	}
	if embed.Thumbnail == nil || embed.Thumbnail.URL != "https://i.ytimg.com/thumb.jpg" {
		t.Errorf("thumbnail = %+v, want track thumbnail", embed.Thumbnail)
	}

	progressIdx, controlsIdx := -1, -1
	for i, f := range embed.Fields {
		if strings.Contains(f.Value, "🔵") {
			progressIdx = i
		}
		if strings.Contains(f.Value, "⏯") {
			controlsIdx = i
		}
	}
	if progressIdx == -1 {
		t.Fatal("no progress bar field found")
	}
	if controlsIdx == -1 {
		t.Fatal("no controls field found")
	}
	if controlsIdx < progressIdx {
		t.Errorf("controls field (%d) must be beneath the progress field (%d)", controlsIdx, progressIdx)
	}

	progress := embed.Fields[progressIdx].Value
	if !strings.Contains(progress, "0:35") || !strings.Contains(progress, "4:18") {
		t.Errorf("progress field %q missing elapsed/total", progress)
	}

	controls := embed.Fields[controlsIdx].Value
	if !strings.Contains(controls, "⏮") || !strings.Contains(controls, "⏭") {
		t.Errorf("controls field %q missing prev/next icons", controls)
	}
}

func TestBuildNowPlayingPanelSpotifyFooter(t *testing.T) {
	status := &services.PlaybackStatus{Track: &services.Track{Title: "S", Duration: 100, Source: "spotify"}}
	embeds := discord.BuildNowPlayingPanel(status, time.Now())
	if embeds[0].Footer == nil || embeds[0].Footer.Text != "Spotify • Mixtape" {
		t.Errorf("footer = %+v, want Spotify • Mixtape", embeds[0].Footer)
	}
}

func TestBuildNowPlayingPanelBarPositions(t *testing.T) {
	track := &services.Track{Title: "Song", Duration: 150, Source: "youtube"}

	cases := []struct {
		name    string
		elapsed time.Duration
		wantDot int // rune position of 🔵 within the 15 segments
	}{
		{"start", 0, 0},
		{"mid", 75 * time.Second, 7},
		{"end", 150 * time.Second, 14},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			embeds := discord.BuildNowPlayingPanel(&services.PlaybackStatus{Track: track, Elapsed: tc.elapsed}, time.Now())
			var bar string
			for _, f := range embeds[0].Fields {
				if strings.Contains(f.Value, "▬") || strings.Contains(f.Value, "🔵") {
					bar = strings.Split(f.Value, " ")[1]
				}
			}

			runes := []rune(bar)
			dot := strings.IndexRune(bar, '🔵')
			if dot == -1 {
				t.Fatalf("bar %q has no dot", bar)
			}
			if got := runeIndex(bar, '🔵'); got != tc.wantDot {
				t.Errorf("dot at rune %d, want %d (bar %q)", got, tc.wantDot, bar)
			}
			if len(runes) != 15 {
				t.Errorf("bar %q has %d runes, want 15", bar, len(runes))
			}
		})
	}
}

// runeIndex returns the rune index of r in s, or -1.
func runeIndex(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return len([]rune(s[:i]))
		}
	}
	return -1
}

func TestBuildNowPlayingPanelUnknownDuration(t *testing.T) {
	track := &services.Track{Title: "Live", Source: "youtube"}
	embeds := discord.BuildNowPlayingPanel(&services.PlaybackStatus{Track: track, Elapsed: 0}, time.Now())

	var bar string
	for _, f := range embeds[0].Fields {
		if strings.Contains(f.Value, "??:??") {
			bar = f.Value
		}
	}
	if bar == "" {
		t.Fatal("want ??:?? placeholder for unknown duration")
	}
	if strings.Contains(bar, "🔵") {
		t.Errorf("bar %q should have no dot when duration unknown", bar)
	}
}

func TestBuildNowPlayingPanelNoThumbnail(t *testing.T) {
	track := &services.Track{Title: "Song", Duration: 100, Source: "youtube"}
	embeds := discord.BuildNowPlayingPanel(&services.PlaybackStatus{Track: track}, time.Now())
	if embeds[0].Thumbnail != nil {
		t.Errorf("thumbnail = %+v, want nil when track has no thumbnail", embeds[0].Thumbnail)
	}
}

func TestBuildNowPlayingPanelIsPurelyVisual(t *testing.T) {
	// The player card is a pure embed snapshot: the panel builder must not
	// produce interactive components (controls are icons inside the card).
	// The command therefore responds with embeds only — asserted via the
	// builder returning no component data at all.
	embeds := discord.BuildNowPlayingPanel(playingStatus(0), time.Now())
	if len(embeds) != 1 {
		t.Fatalf("want exactly one visual embed, got %d", len(embeds))
	}
}
