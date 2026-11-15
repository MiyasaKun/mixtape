package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"fiber_test/internal/domain"
)

const (
	defaultAccountsBase = "https://accounts.spotify.com"
	defaultAPIBase      = "https://api.spotify.com/v1"

	// Read-only: enough to list a user's private and collaborative
	// playlists, their Liked Songs, plus the profile used for display.
	spotifyScopes = "playlist-read-private playlist-read-collaborative user-read-private user-library-read"

	maxPlaylistLimit = 50
	maxTrackLimit    = 100
)

// Option customises a Spotify adapter.
type Option func(*Spotify)

// WithEndpoints overrides the Spotify accounts and Web API base URLs. It is
// intended for tests; production always uses Spotify's own hosts.
func WithEndpoints(accountsBase, apiBase string) Option {
	return func(s *Spotify) {
		s.accountsBase = accountsBase
		s.apiBase = apiBase
	}
}

type Spotify struct {
	clientID     string
	clientSecret string
	redirectURI  string
	httpClient   *http.Client
	accountsBase string
	apiBase      string

	// appToken caches a client-credentials access token for app-level API
	// calls (e.g. track lookups that need no user consent).
	appToken     string
	appTokenExps time.Time
	appTokenMu   sync.Mutex
}

func New(clientID, clientSecret, redirectURI string, opts ...Option) *Spotify {
	s := &Spotify{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		accountsBase: defaultAccountsBase,
		apiBase:      defaultAPIBase,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Spotify) tokenEndpoint() string     { return s.accountsBase + "/api/token" }
func (s *Spotify) authorizeEndpoint() string { return s.accountsBase + "/authorize" }

// AuthURL builds the Spotify consent URL. State is the caller's anti-CSRF
// value.
func (s *Spotify) AuthURL(state string) string {
	params := url.Values{
		"client_id":     {s.clientID},
		"redirect_uri":  {s.redirectURI},
		"response_type": {"code"},
		"scope":         {spotifyScopes},
		"state":         {state},
		"show_dialog":   {"false"},
	}
	return s.authorizeEndpoint() + "?" + params.Encode()
}

// ExchangeCode trades an authorization code for tokens and the Spotify
// profile they belong to.
func (s *Spotify) ExchangeCode(ctx context.Context, code, redirectURI string) (domain.SpotifyTokens, domain.SpotifyProfile, error) {
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
	}

	body, err := s.postToken(ctx, form)
	if err != nil {
		return domain.SpotifyTokens{}, domain.SpotifyProfile{}, err
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.SpotifyTokens{}, domain.SpotifyProfile{}, fmt.Errorf("decode token response: %w", err)
	}
	if payload.RefreshToken == "" {
		return domain.SpotifyTokens{}, domain.SpotifyProfile{}, fmt.Errorf("spotify token exchange returned no refresh token")
	}

	tokens := domain.SpotifyTokens{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second),
	}

	profile, err := s.profile(ctx, payload.AccessToken)
	if err != nil {
		return domain.SpotifyTokens{}, domain.SpotifyProfile{}, err
	}
	return tokens, profile, nil
}

// RefreshAccessToken trades a refresh token for a fresh access token. Spotify
// may omit a new refresh token; callers should keep the existing one when it is
// empty.
func (s *Spotify) RefreshAccessToken(ctx context.Context, refreshToken string) (domain.SpotifyTokens, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return domain.SpotifyTokens{}, fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.clientID, s.clientSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return domain.SpotifyTokens{}, fmt.Errorf("spotify token refresh: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusBadRequest {
		return domain.SpotifyTokens{}, domain.ErrSpotifyReconnectRequired
	}
	if resp.StatusCode != http.StatusOK {
		return domain.SpotifyTokens{}, fmt.Errorf("spotify token refresh failed: status %d: %s", resp.StatusCode, body)
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.SpotifyTokens{}, fmt.Errorf("decode refresh response: %w", err)
	}
	if payload.AccessToken == "" {
		return domain.SpotifyTokens{}, fmt.Errorf("spotify token refresh returned no access token")
	}

	return domain.SpotifyTokens{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second),
	}, nil
}

// GetTrack fetches a single track by its Spotify ID. It authenticates with a
// client-credentials token (cached until close to expiry), so no user OAuth
// connection is required for track lookups.
func (s *Spotify) GetTrack(ctx context.Context, trackID string) (domain.PlaylistTrack, error) {
	accessToken, err := s.appAccessToken(ctx)
	if err != nil {
		return domain.PlaylistTrack{}, err
	}

	var payload spotifyTrack
	if err := s.get(ctx, accessToken, s.apiBase+"/tracks/"+url.PathEscape(trackID), nil, &payload); err != nil {
		return domain.PlaylistTrack{}, err
	}
	return mapTrack(&payload), nil
}

// appAccessToken returns a valid client-credentials access token, fetching a
// new one when none is cached or the cached one is about to expire.
func (s *Spotify) appAccessToken(ctx context.Context) (string, error) {
	s.appTokenMu.Lock()
	defer s.appTokenMu.Unlock()

	if s.appToken != "" && time.Now().Before(s.appTokenExps) {
		return s.appToken, nil
	}

	body, err := s.postToken(ctx, url.Values{"grant_type": {"client_credentials"}})
	if err != nil {
		return "", err
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("spotify token exchange returned no access token")
	}

	s.appToken = payload.AccessToken
	s.appTokenExps = time.Now().Add(time.Duration(payload.ExpiresIn)*time.Second - 30*time.Second)
	return s.appToken, nil
}

// ListPlaylists returns a page of the user's followed and owned playlists.
func (s *Spotify) ListPlaylists(ctx context.Context, accessToken, spotifyUserID string, offset, limit int) (domain.PlaylistPage, error) {
	offset = clampOffset(offset)
	limit = clampLimit(limit, maxPlaylistLimit)

	var payload struct {
		Items []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Description   string `json:"description"`
			Collaborative bool   `json:"collaborative"`
			Owner         struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
			} `json:"owner"`
			// Spotify renamed the playlist contents field from "tracks" to
			// "items" in February 2026. Its presence is the signal that the app
			// may read the playlist's contents: under Development Mode it is
			// only returned for playlists the user owns or collaborates on,
			// so both fields are pointers.
			Items *struct {
				Total int `json:"total"`
			} `json:"items"`
			Tracks *struct {
				Total int `json:"total"`
			} `json:"tracks"`
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
			ExternalURLs struct {
				Spotify string `json:"spotify"`
			} `json:"external_urls"`
		} `json:"items"`
		Next string `json:"next"`
	}

	query := url.Values{
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}
	if err := s.get(ctx, accessToken, s.apiBase+"/me/playlists", query, &payload); err != nil {
		return domain.PlaylistPage{}, err
	}

	items := make([]domain.Playlist, 0, len(payload.Items))
	for _, p := range payload.Items {
		image := ""
		if len(p.Images) > 0 {
			image = p.Images[0].URL
		}
		trackTotal := 0
		browseable := p.Items != nil
		switch {
		case p.Items != nil:
			trackTotal = p.Items.Total
		case p.Tracks != nil:
			trackTotal = p.Tracks.Total
		}
		items = append(items, domain.Playlist{
			ID:            p.ID,
			Name:          p.Name,
			Description:   p.Description,
			Owner:         p.Owner.DisplayName,
			TrackCount:    trackTotal,
			ImageURL:      image,
			URL:           p.ExternalURLs.Spotify,
			IsOwned:       spotifyUserID != "" && p.Owner.ID == spotifyUserID,
			Collaborative: p.Collaborative,
			Browseable:    browseable,
		})
	}

	return domain.PlaylistPage{Items: items, Next: nextOffset(payload.Next, offset, len(items))}, nil
}

// ListLikedTracks returns a page of the user's Liked Songs.
func (s *Spotify) ListLikedTracks(ctx context.Context, accessToken string, offset, limit int) (domain.TrackPage, error) {
	offset = clampOffset(offset)
	limit = clampLimit(limit, maxTrackLimit)

	var payload struct {
		Items []playlistItem `json:"items"`
		Next  string         `json:"next"`
	}

	query := url.Values{
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}
	if err := s.get(ctx, accessToken, s.apiBase+"/me/tracks", query, &payload); err != nil {
		return domain.TrackPage{}, err
	}

	items := make([]domain.PlaylistTrack, 0, len(payload.Items))
	for _, saved := range payload.Items {
		if track := saved.song(); track != nil {
			items = append(items, mapTrack(track))
		}
	}

	return domain.TrackPage{Items: items, Next: nextOffset(payload.Next, offset, len(items))}, nil
}

// ListPlaylistTracks returns a page of tracks in a single playlist. Spotify
// only returns the contents of playlists the user owns or collaborates on;
// other playlists yield an empty page.
func (s *Spotify) ListPlaylistTracks(ctx context.Context, accessToken, playlistID string, offset, limit int) (domain.TrackPage, error) {
	offset = clampOffset(offset)
	limit = clampLimit(limit, maxTrackLimit)

	var payload struct {
		Items []playlistItem `json:"items"`
		Next  string         `json:"next"`
	}

	query := url.Values{
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}
	// February 2026: /playlists/{id}/tracks was replaced by /items.
	endpoint := s.apiBase + "/playlists/" + url.PathEscape(playlistID) + "/items"
	if err := s.get(ctx, accessToken, endpoint, query, &payload); err != nil {
		return domain.TrackPage{}, err
	}

	items := make([]domain.PlaylistTrack, 0, len(payload.Items))
	for _, saved := range payload.Items {
		if track := saved.song(); track != nil {
			items = append(items, mapTrack(track))
		}
	}

	return domain.TrackPage{Items: items, Next: nextOffset(payload.Next, offset, len(items))}, nil
}

// playlistItem is one entry in a playlist or liked-songs page. Spotify renamed
// the nested "track" field to "item" in February 2026; both are read so the
// adapter keeps working across the change.
type playlistItem struct {
	Item  *spotifyTrack `json:"item"`
	Track *spotifyTrack `json:"track"`
}

func (p playlistItem) song() *spotifyTrack {
	if p.Item != nil {
		return p.Item
	}
	return p.Track
}

// postToken performs a client-credentials-authenticated form POST to the token
// endpoint and returns the raw response body on success.
func (s *Spotify) postToken(ctx context.Context, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.clientID, s.clientSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spotify token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify token exchange failed: status %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

// get performs an authenticated GET and decodes the JSON body into out.
func (s *Spotify) get(ctx context.Context, accessToken, endpoint string, query url.Values, out any) error {
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build spotify request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("spotify request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return &domain.SpotifyAPIError{StatusCode: resp.StatusCode, Operation: "request", Body: string(body)}
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode spotify response: %w", err)
	}
	return nil
}

func (s *Spotify) profile(ctx context.Context, accessToken string) (domain.SpotifyProfile, error) {
	var payload struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	if err := s.get(ctx, accessToken, s.apiBase+"/me", nil, &payload); err != nil {
		return domain.SpotifyProfile{}, err
	}
	if payload.ID == "" {
		return domain.SpotifyProfile{}, fmt.Errorf("spotify profile returned no id")
	}
	return domain.SpotifyProfile{ID: payload.ID, DisplayName: payload.DisplayName}, nil
}

type spotifyTrack struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	DurationMS int    `json:"duration_ms"`
	Artists    []struct {
		Name string `json:"name"`
	} `json:"artists"`
	Album struct {
		Name   string `json:"name"`
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	} `json:"album"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

func mapTrack(t *spotifyTrack) domain.PlaylistTrack {
	artists := make([]string, 0, len(t.Artists))
	for _, a := range t.Artists {
		if a.Name != "" {
			artists = append(artists, a.Name)
		}
	}
	image := ""
	if len(t.Album.Images) > 0 {
		image = t.Album.Images[0].URL
	}
	return domain.PlaylistTrack{
		ID:         t.ID,
		Title:      t.Name,
		Artists:    artists,
		Album:      t.Album.Name,
		DurationMS: t.DurationMS,
		ImageURL:   image,
		URL:        t.ExternalURLs.Spotify,
	}
}

// nextOffset derives the next page offset from Spotify's `next` URL. It falls
// back to offset+count when the URL cannot be parsed.
func nextOffset(next string, offset, count int) *int {
	if next == "" || count == 0 {
		return nil
	}
	if parsed, err := url.Parse(next); err == nil {
		if raw := parsed.Query().Get("offset"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil {
				return &n
			}
		}
	}
	n := offset + count
	return &n
}

func clampOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func clampLimit(limit, max int) int {
	if limit <= 0 {
		return max
	}
	if limit > max {
		return max
	}
	return limit
}
