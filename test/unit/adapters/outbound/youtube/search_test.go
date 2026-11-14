package youtube_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"fiber_test/internal/adapters/outbound/youtube"
)

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// fakeRunner returns a yt-dlp runner stub that records the invoked args and
// answers with the given output.
func fakeRunner(argsOut [][]string, out func(call int) []byte) (func(ctx context.Context, args ...string) ([]byte, error), *[][]string) {
	calls := &argsOut
	call := 0
	return func(ctx context.Context, args ...string) ([]byte, error) {
		*calls = append(*calls, args)
		if out == nil {
			return nil, errors.New("no output configured")
		}
		out(call)
		call++
		return out(call - 1), nil
	}, calls
}

func entry(id, title, channel string, duration int64, url string) map[string]any {
	return map[string]any{
		"id": id, "title": title, "channel": channel,
		"duration": duration, "url": url,
		"thumbnail": "https://img.example/" + id + ".jpg",
	}
}

func TestSearchReadsFlatPlaylistEntryURLField(t *testing.T) {
	// --flat-playlist entries carry the video URL in "url", not "webpage_url".
	var args [][]string
	run, _ := fakeRunner(args, func(int) []byte {
		payload, _ := json.Marshal(map[string]any{
			"entries": []any{
				entry("vid-1", "Nightcall (Official Audio)", "Kavinsky", 258, "https://www.youtube.com/watch?v=vid-1"),
			},
		})
		return payload
	})

	client := youtube.New(youtube.WithRunner(run))
	tracks, err := client.Search(context.Background(), "Kavinsky - Nightcall", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("want 1 track, got %d", len(tracks))
	}
	if tracks[0].URL != "https://www.youtube.com/watch?v=vid-1" {
		t.Errorf("want flat entry url mapped, got %q", tracks[0].URL)
	}
}

func TestSearchStillReadsWebpageURL(t *testing.T) {
	var args [][]string
	run, _ := fakeRunner(args, func(int) []byte {
		payload, _ := json.Marshal(map[string]any{
			"entries": []any{
				map[string]any{"id": "vid-1", "title": "T", "webpage_url": "https://www.youtube.com/watch?v=vid-1"},
			},
		})
		return payload
	})

	client := youtube.New(youtube.WithRunner(run))
	tracks, err := client.Search(context.Background(), "q", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 1 || tracks[0].URL != "https://www.youtube.com/watch?v=vid-1" {
		t.Fatalf("unexpected tracks: %+v", tracks)
	}
}

func TestSearchReturnsYTSearchResults(t *testing.T) {
	var args [][]string
	run, _ := fakeRunner(args, func(int) []byte {
		payload, _ := json.Marshal(map[string]any{
			"entries": []any{
				entry("vid-1", "Nightcall (Official Audio)", "Kavinsky", 258, "https://www.youtube.com/watch?v=vid-1"),
				entry("vid-2", "Nightcall - Live", "Kavinsky TV", 292, "https://www.youtube.com/watch?v=vid-2"),
			},
		})
		return payload
	})

	client := youtube.New(youtube.WithRunner(run))

	tracks, err := client.Search(context.Background(), "Kavinsky - Nightcall", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tracks) != 2 {
		t.Fatalf("want 2 tracks, got %d", len(tracks))
	}
	if tracks[0].ID != "vid-1" || tracks[0].Title != "Nightcall (Official Audio)" {
		t.Errorf("unexpected first track: %+v", tracks[0])
	}
	if tracks[0].Artist != "Kavinsky" || tracks[0].Duration != 258 {
		t.Errorf("unexpected first track metadata: %+v", tracks[0])
	}
	if tracks[0].URL != "https://www.youtube.com/watch?v=vid-1" {
		t.Errorf("want webpage URL, got %q", tracks[0].URL)
	}
	if tracks[1].ID != "vid-2" {
		t.Errorf("unexpected second track: %+v", tracks[1])
	}
}

func TestSearchPassesFlatPlaylistYTSearchQuery(t *testing.T) {
	var args [][]string
	run, calls := fakeRunner(args, func(int) []byte {
		return []byte(`{"entries":[]}`)
	})

	client := youtube.New(youtube.WithRunner(run))
	if _, err := client.Search(context.Background(), "Kavinsky - Nightcall", 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*calls) != 1 {
		t.Fatalf("want 1 yt-dlp call, got %d", len(*calls))
	}
	got := fmt.Sprint((*calls)[0])
	if !contains(got, "ytsearch5:Kavinsky - Nightcall") {
		t.Errorf("want ytsearch5 query in args, got %s", got)
	}
	if !contains(got, "--flat-playlist") {
		t.Errorf("want --flat-playlist in args, got %s", got)
	}
}

func TestSearchHandlesMissingEntriesAndNoResults(t *testing.T) {	var args [][]string
	run, _ := fakeRunner(args, func(int) []byte {
		return []byte(`{"entries":null}`)
	})

	client := youtube.New(youtube.WithRunner(run))
	tracks, err := client.Search(context.Background(), "nothing here", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 0 {
		t.Errorf("want 0 tracks, got %d", len(tracks))
	}
}

func TestSearchSurfacesRunnerErrors(t *testing.T) {
	client := youtube.New(youtube.WithRunner(func(ctx context.Context, args ...string) ([]byte, error) {
		return nil, errors.New("boom")
	}))

	if _, err := client.Search(context.Background(), "q", 5); err == nil {
		t.Fatal("want error from failing runner")
	}
}
