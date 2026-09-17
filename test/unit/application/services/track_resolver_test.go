package services_test

import (
	"context"
	"strings"
	"testing"

	"fiber_test/internal/application/services"
)

func TestTrackResolverRejectsUnsupportedSources(t *testing.T) {
	resolver := services.NewTrackResolverService(nil, nil, nil)

	tests := []string{
		"https://open.spotify.com/track/example",
		"https://soundcloud.com/artist/example",
	}
	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			_, err := resolver.Resolve(context.Background(), query)
			if err == nil {
				t.Fatal("Resolve() returned nil error")
			}
			if !strings.Contains(err.Error(), "not yet supported") {
				t.Fatalf("Resolve() error = %q", err)
			}
		})
	}
}
