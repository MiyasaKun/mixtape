package discord

import (
	"fmt"
	"math"
	"strings"
	"time"

	"fiber_test/internal/application/services"

	dgo "github.com/disgoorg/disgo/discord"
)

// nowPlayingBarSegments is the number of ▬ segments in the progress bar.
// ▬ and 🔵 are used because they render reliably in Discord's embed font
// (━ tends to get substituted with broken glyphs on some clients).
const nowPlayingBarSegments = 15

// nowPlayingControls is the purely visual control row rendered inside the
// card beneath the progress bar. These are icons, not Discord buttons.
const nowPlayingControls = "⏮　⏯　⏭"

// sourceLabel maps a Track.Source value to a display label.
var sourceLabel = map[string]string{
	"youtube": "YouTube",
	"spotify": "Spotify",
}

// BuildNowPlayingPanel renders a snapshot of the current playback as a
// player-like embed: cover, title/artist, progress bar, and a purely visual
// control row. takenAt is the moment the snapshot was captured; it is
// reserved for future live-update flows and currently unused by rendering.
func BuildNowPlayingPanel(status *services.PlaybackStatus, takenAt time.Time) []dgo.Embed {
	_ = takenAt

	track := status.Track

	description := "**" + track.Title + "**"
	if track.Artist != "" {
		description += "\nby " + track.Artist
	}

	source := track.Source
	if label, ok := sourceLabel[source]; ok {
		source = label
	}

	embed := dgo.Embed{
		Title:       "🎵 Now Playing",
		Description: description,
		Color:       0x5865F2,
		Footer:      &dgo.EmbedFooter{Text: source + " • Mixtape"},
		Thumbnail:   nil,
	}
	if track.ThumbnailURL != "" {
		embed.Thumbnail = &dgo.EmbedResource{URL: track.ThumbnailURL}
	}
	embed.Fields = []dgo.EmbedField{
		{Name: "Progress", Value: nowPlayingBar(status.Elapsed, track.Duration)},
		// The control row lives inside the card; the icons have no function.
		{Name: "\u200b", Value: nowPlayingControls},
	}
	return []dgo.Embed{embed}
}

// nowPlayingBar renders the progress bar: "0:35 ▬▬🔵▬▬ 4:18". With 15
// segments the 🔵 sits at round(elapsed/total×14). Unknown durations render
// an empty bar with a "??:??" placeholder on both sides.
func nowPlayingBar(elapsed time.Duration, totalSeconds int64) string {
	if totalSeconds <= 0 {
		return fmt.Sprintf("%s %s %s", "??:??", strings.Repeat("▬", nowPlayingBarSegments), "??:??")
	}

	total := time.Duration(totalSeconds) * time.Second
	if elapsed > total {
		elapsed = total
	}
	dot := int(math.Round(float64(elapsed) / float64(total) * float64(nowPlayingBarSegments-1)))
	if dot < 0 {
		dot = 0
	}
	if dot >= nowPlayingBarSegments {
		dot = nowPlayingBarSegments - 1
	}

	segments := []rune(strings.Repeat("▬", nowPlayingBarSegments))
	bar := string(segments[:dot]) + "🔵" + string(segments[dot+1:])

	return fmt.Sprintf("%s %s %s", formatDuration(int64(elapsed/time.Second)), bar, formatDuration(totalSeconds))
}

// ParseNowPlayingButton was removed with the interactive control buttons:
// the player card is purely visual (icons rendered inside the embed).
