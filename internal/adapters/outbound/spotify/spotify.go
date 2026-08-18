package spotify

type Spotify struct {
	clientID     string
	clientSecret string
	redirectURI  string
}

func New(clientID, clientSecret, redirectURI string) *Spotify {
	return &Spotify{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
	}
}
