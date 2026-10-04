package domain

// Playlist is a Spotify playlist shown in the playlists browser. It is a
// summary: tracks are fetched separately, page by page.
type Playlist struct {
	ID            string
	Name          string
	Description   string
	Owner         string
	TrackCount    int
	ImageURL      string
	URL           string
	IsOwned       bool
	Collaborative bool
	// Browseable is true when Spotify exposes this playlist's contents to the
	// app. Under Development Mode only playlists the user owns or collaborates
	// on are readable; others return metadata only.
	Browseable bool
}

// PlaylistTrack is a single song inside a playlist or the Liked Songs library.
type PlaylistTrack struct {
	ID         string
	Title      string
	Artists    []string
	Album      string
	DurationMS int
	ImageURL   string
	URL        string
}

// PlaylistPage is one page of playlists. Next is the offset to request for the
// following page, or nil when the list is exhausted.
type PlaylistPage struct {
	Items []Playlist
	Next  *int
}

// TrackPage is one page of playlist tracks. Next is the offset to request for
// the following page, or nil when the list is exhausted.
type TrackPage struct {
	Items []PlaylistTrack
	Next  *int
}
