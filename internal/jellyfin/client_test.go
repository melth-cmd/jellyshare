package jellyfin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestContainsEpisode(t *testing.T) {
	episodes := []EpisodeInfo{{ID: "a"}, {ID: "b"}}
	for id, want := range map[string]bool{"a": true, "b": true, "c": false, "": false} {
		if got := ContainsEpisode(episodes, id); got != want {
			t.Errorf("%q: got %v, want %v", id, got, want)
		}
	}
	if ContainsEpisode(nil, "a") {
		t.Error("an empty list contains nothing")
	}
}

// Jellyfin leaves an item's overview out of a list unless the request names
// it, so the episode list a viewer sees has to ask for it - and the id-only
// checks must not ask for anything.
func TestEpisodeQueriesAskOnlyForTheFieldsTheyNeed(t *testing.T) {
	var fields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Like Jellyfin, only the fields asked for come back.
		fields = r.URL.Query().Get("Fields")
		item := `{"Id":"e1","Name":"Earthfall","Type":"Episode"`
		if strings.Contains(fields, "Overview") {
			item += `,"Overview":"An object from space."`
		}
		if strings.Contains(fields, "MediaSources") {
			item += `,"MediaSources":[{"Size":42}]`
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Items":[` + item + `}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	c.userID = "u"
	episodes, err := c.GetEpisodeListFor(context.Background(), "Season", "s1")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Overview", "MediaSources"} {
		if !strings.Contains(fields, want) {
			t.Errorf("Fields=%q should ask for %s", fields, want)
		}
	}
	if len(episodes) != 1 || episodes[0].Overview != "An object from space." || episodes[0].SizeBytes != 42 {
		t.Errorf("got %+v", episodes)
	}

	// The checks that only need the ids ask Jellyfin for no extra fields; media
	// sources alone cost it a lookup per episode.
	if _, err := c.GetEpisodesFor(context.Background(), "Season", "s1"); err != nil {
		t.Fatal(err)
	}
	if fields != "" {
		t.Errorf("GetEpisodesFor asked for Fields=%q, want none", fields)
	}
}

func TestSearchItemsQueriesLibraryAndMapsResults(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		if r.URL.Path != "/Users/u/Items" {
			t.Errorf("path = %q, want /Users/u/Items", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Items":[
			{"Id":"m1","Name":"Alien","Type":"Movie","ProductionYear":1979,"RunTimeTicks":70200000000,"ImageTags":{"Primary":"tag"}},
			{"Id":"s1","Name":"Alien","Type":"Series","ProductionYear":2025}
		]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	c.userID = "u"
	items, err := c.SearchItems(context.Background(), "alien", 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("SearchTerm") != "alien" {
		t.Errorf("SearchTerm = %q, want alien", got.Get("SearchTerm"))
	}
	if got.Get("Limit") != "7" {
		t.Errorf("Limit = %q, want 7", got.Get("Limit"))
	}
	if got.Get("SortBy") != "SortName" {
		t.Errorf("SortBy = %q, want SortName", got.Get("SortBy"))
	}
	if types := got.Get("IncludeItemTypes"); types != "Movie,Series" {
		t.Errorf("IncludeItemTypes = %q, want only top-level Movie,Series", types)
	}
	if fields := got.Get("Fields"); !strings.Contains(fields, "Overview") || !strings.Contains(fields, "ImageTags") {
		t.Errorf("Fields = %q", fields)
	}

	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Year != 1979 || !items[0].HasPoster || items[0].RuntimeSeconds != 7020 {
		t.Errorf("movie mapped wrong: %+v", items[0])
	}
	if items[1].Type != "Series" || items[1].Year != 2025 {
		t.Errorf("series mapped wrong: %+v", items[1])
	}
}

func TestSearchItemsEmptyQueryListsRecentItems(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Items":[]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	c.userID = "u"
	items, err := c.SearchItems(context.Background(), "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want none", len(items))
	}
	if got.Get("SearchTerm") != "" {
		t.Errorf("SearchTerm = %q, want none", got.Get("SearchTerm"))
	}
	if got.Get("SortBy") != "DateCreated" || got.Get("SortOrder") != "Descending" {
		t.Errorf("SortBy/SortOrder = %q/%q", got.Get("SortBy"), got.Get("SortOrder"))
	}
	if got.Get("Limit") != "20" {
		t.Errorf("Limit = %q, want the default 20", got.Get("Limit"))
	}
}

func TestSearchItemsReportsJellyfinErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	c.userID = "u"
	if _, err := c.SearchItems(context.Background(), "alien", 5); err == nil {
		t.Fatal("expected an error")
	}
}

func TestBrowseSeasonsAndEpisodes(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		switch got.Get("IncludeItemTypes") {
		case "Season":
			w.Write([]byte(`{"Items":[
				{"Id":"se0","Name":"Specials","Type":"Season","IndexNumber":0},
				{"Id":"se1","Name":"Season 1","Type":"Season","IndexNumber":1,"ImageTags":{"Primary":"tag"}}
			]}`))
		case "Episode":
			w.Write([]byte(`{"Items":[
				{"Id":"e1","Name":"Pilot","Type":"Episode","SeriesName":"Alien","ParentIndexNumber":1,"IndexNumber":1,"RunTimeTicks":3000000000}
			]}`))
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	c.userID = "u"

	seasons, err := c.BrowseSeasons(context.Background(), "series1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("ParentId") != "series1" || got.Get("SortBy") != "IndexNumber" || got.Get("Recursive") != "" {
		t.Errorf("season query = %v", got)
	}
	if len(seasons) != 2 || seasons[0].SeasonNumber != 0 || seasons[1].SeasonNumber != 1 || !seasons[1].HasPoster {
		t.Errorf("seasons mapped wrong: %+v", seasons)
	}

	episodes, err := c.BrowseEpisodes(context.Background(), "season1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("ParentId") != "season1" || got.Get("SortBy") != "IndexNumber" {
		t.Errorf("episode query = %v", got)
	}
	if len(episodes) != 1 || episodes[0].SeasonNumber != 1 || episodes[0].EpisodeNumber != 1 || episodes[0].RuntimeSeconds != 300 {
		t.Errorf("episode mapped wrong: %+v", episodes)
	}

	if _, err := c.BrowseSeriesEpisodes(context.Background(), "series1"); err != nil {
		t.Fatal(err)
	}
	if got.Get("ParentId") != "series1" || got.Get("Recursive") != "true" || got.Get("SortBy") != "ParentIndexNumber,IndexNumber" {
		t.Errorf("fallback episode query = %v", got)
	}
}
