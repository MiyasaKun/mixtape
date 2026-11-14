package discord

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"fiber_test/internal/application/services"

	dgo "github.com/disgoorg/disgo/discord"
)

// spotifyPickPrefix is the custom-ID prefix of the Spotify track-pick select
// menu. The full ID is spotifyPickPrefix:<guildID>:<userID>:<requestID>:<n>,
// where n is the menu's default option index (unused for matching).
const spotifyPickPrefix = "spotifypick"

// SelectionTTL is how long a posted select menu stays valid.
const SelectionTTL = time.Minute

// SelectionStore keeps the candidate tracks behind a pending Spotify select
// menu, keyed by a one-time request ID. Entries expire after the TTL so a
// menu nobody used cannot leak tracks.
type SelectionStore struct {
	mu   sync.Mutex
	ttl  time.Duration
	next map[string]selectionEntry
}

type selectionEntry struct {
	tracks    []services.Track
	requester string
	guildID   string
	// coverURL is the Spotify album image of the track behind the pick, if
	// the pick came from a Spotify link; "" otherwise.
	coverURL  string
	expiresAt time.Time
}

func NewSelectionStore(ttl time.Duration) *SelectionStore {
	return &SelectionStore{ttl: ttl, next: make(map[string]selectionEntry)}
}

// Put stores the candidates under a fresh one-time request ID. coverURL is
// the Spotify album image for tracks that came from a Spotify link (""
// otherwise) and is handed back by Take.
func (s *SelectionStore) Put(requester, guildID, coverURL string, tracks []services.Track) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()

	id, err := randomRequestID()
	if err != nil {
		// Fall back to a monotonic-ish ID; collisions only cost one lookup.
		id = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	s.next[id] = selectionEntry{
		tracks:    tracks,
		requester: requester,
		guildID:   guildID,
		coverURL:  coverURL,
		expiresAt: time.Now().Add(s.ttl),
	}
	return id
}

// Take returns and removes the candidates for the request ID, together with
// the stored cover URL. It misses for unknown, expired, or already-used IDs.
func (s *SelectionStore) Take(requestID string) ([]services.Track, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()

	entry, ok := s.next[requestID]
	if !ok {
		return nil, "", false
	}
	delete(s.next, requestID)
	return entry.tracks, entry.coverURL, true
}

// sweepLocked drops expired entries. The caller must hold the mutex.
func (s *SelectionStore) sweepLocked() {
	now := time.Now()
	for id, entry := range s.next {
		if now.After(entry.expiresAt) {
			delete(s.next, id)
		}
	}
}

func randomRequestID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// SpotifyPickCustomID builds the select menu custom ID for a Spotify pick.
func SpotifyPickCustomID(guildID, userID, requestID string, index int) string {
	return fmt.Sprintf("%s:%s:%s:%s:%d", spotifyPickPrefix, guildID, userID, requestID, index)
}

// ParseSpotifyPickCustomID splits a custom ID into its routing parts. ok is
// false for custom IDs that do not belong to the Spotify pick flow.
func ParseSpotifyPickCustomID(customID string) (guildID, userID, requestID string, index int, ok bool) {
	parts := strings.Split(customID, ":")
	if len(parts) != 5 || parts[0] != spotifyPickPrefix {
		return "", "", "", 0, false
	}
	i, err := fmt.Sscanf(parts[4], "%d", &index)
	if err != nil || i != 1 {
		return "", "", "", 0, false
	}
	return parts[1], parts[2], parts[3], index, true
}

// BuildSpotifyPickButtons builds one button per candidate, labeled 1..n. The
// button's custom ID encodes the candidate index; the numbered list
// (SpotifyPickList) is posted alongside so users see what each button means.
func BuildSpotifyPickButtons(guildID, userID, requestID string, candidates []services.Track) []dgo.InteractiveComponent {
	buttons := make([]dgo.InteractiveComponent, 0, len(candidates))
	for i := range candidates {
		buttons = append(buttons, dgo.NewPrimaryButton(
			fmt.Sprintf("%d", i+1),
			SpotifyPickCustomID(guildID, userID, requestID, i),
		))
	}
	return buttons
}

// SpotifySelectMenuComponent is the layout component of the pick buttons,
// ready to be embedded in a MessageCreate.
func SpotifySelectMenuComponent(guildID, userID, requestID string, candidates []services.Track) []dgo.LayoutComponent {
	buttons := BuildSpotifyPickButtons(guildID, userID, requestID, candidates)
	return []dgo.LayoutComponent{dgo.NewActionRow(buttons...)}
}

// SpotifyPickList renders the candidates as a numbered list matching the
// pick buttons: "1. Artist – Title (m:ss)".
func SpotifyPickList(candidates []services.Track) string {
	var b strings.Builder
	for i, candidate := range candidates {
		fmt.Fprintf(&b, "%d. %s\n", i+1, candidateLabel(candidate))
	}
	return b.String()
}

// candidateLabel renders a candidate as "Artist – Title (m:ss)".
func candidateLabel(track services.Track) string {
	var b strings.Builder
	if track.Artist != "" {
		b.WriteString(track.Artist)
		b.WriteString(" – ")
	}
	b.WriteString(track.Title)
	if track.Duration > 0 {
		b.WriteString(fmt.Sprintf(" (%s)", formatDuration(track.Duration)))
	}
	// Discord caps option labels at 100 characters.
	label := b.String()
	if len(label) > 100 {
		label = label[:97] + "..."
	}
	return label
}

func formatDuration(seconds int64) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
