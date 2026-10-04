package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fiber_test/internal/domain"
)

const (
	authEndpoint     = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenEndpoint    = "https://oauth2.googleapis.com/token"
	userInfoEndpoint = "https://openidconnect.googleapis.com/v1/userinfo"
	logoutEndpoint   = "https://accounts.google.com/logout"
)

// Google is the Google OAuth 2.0 client used for sign-in.
type Google struct {
	clientID     string
	clientSecret string
	redirectURI  string
	httpClient   *http.Client
}

func New(clientID, clientSecret, redirectURI string) *Google {
	return &Google{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// AuthURL builds the consent URL. State is the caller's anti-CSRF value.
func (g *Google) AuthURL(state string) string {
	params := url.Values{
		"client_id":     {g.clientID},
		"redirect_uri":  {g.redirectURI},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"access_type":   {"online"},
		"prompt":        {"select_account"},
	}
	return authEndpoint + "?" + params.Encode()
}

// ExchangeCode trades an authorization code for the signer's profile.
func (g *Google) ExchangeCode(ctx context.Context, code, redirectURI string) (domain.GoogleProfile, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	accessToken, err := g.token(ctx, form)
	if err != nil {
		return domain.GoogleProfile{}, err
	}
	return g.profile(ctx, accessToken)
}

func (g *Google) token(ctx context.Context, form url.Values) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("google token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google token exchange failed: status %d: %s", resp.StatusCode, body)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("google token exchange returned no access token")
	}
	return payload.AccessToken, nil
}

func (g *Google) profile(ctx context.Context, accessToken string) (domain.GoogleProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoEndpoint, nil)
	if err != nil {
		return domain.GoogleProfile{}, fmt.Errorf("build userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return domain.GoogleProfile{}, fmt.Errorf("google userinfo: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return domain.GoogleProfile{}, fmt.Errorf("google userinfo failed: status %d: %s", resp.StatusCode, body)
	}

	var payload struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.GoogleProfile{}, fmt.Errorf("decode userinfo: %w", err)
	}
	if payload.Sub == "" {
		return domain.GoogleProfile{}, fmt.Errorf("google userinfo returned no subject")
	}

	return domain.GoogleProfile{
		GoogleID: payload.Sub,
		Email:    payload.Email,
		Name:     payload.Name,
		Picture:  payload.Picture,
	}, nil
}

// SignOutURL returns the Google logout endpoint. Google ignores unknown
// redirect targets, so the local redirect is not forwarded here.
func (g *Google) SignOutURL(redirectURI string) string {
	return logoutEndpoint
}
