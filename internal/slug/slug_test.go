package slug

import (
	"strings"
	"testing"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		name string
		item Item
		want string
	}{
		{
			name: "film",
			item: Item{Type: "Movie", Name: "Obsession"},
			want: "obsession",
		},
		{
			name: "punctuation and spacing",
			item: Item{Type: "Movie", Name: "Mr. & Mrs. Smith!"},
			want: "mr-mrs-smith",
		},
		{
			name: "accents fold to ascii",
			item: Item{Type: "Movie", Name: "Amélie"},
			want: "amelie",
		},
		{
			name: "ligatures expand",
			item: Item{Type: "Movie", Name: "Æon Flux"},
			want: "aeon-flux",
		},
		{
			name: "series",
			item: Item{Type: "Series", Name: "Mr. & Mrs. Smith"},
			want: "mr-mrs-smith",
		},
		{
			name: "season uses the series name and number",
			item: Item{Type: "Season", Name: "Season 1", SeriesName: "Mr. & Mrs. Smith", IndexNumber: 1},
			want: "mr-mrs-smith-s01",
		},
		{
			name: "episode uses series, season and episode numbers",
			item: Item{Type: "Episode", Name: "A Breakup", SeriesName: "Mr. & Mrs. Smith", ParentIndexNumber: 1, IndexNumber: 2},
			want: "mr-mrs-smith-s01-ep-02",
		},
		{
			name: "season without a series name falls back to its own name",
			item: Item{Type: "Season", Name: "Season 2", IndexNumber: 2},
			want: "season-2-s02",
		},
		{
			name: "thai-only title falls back to share",
			item: Item{Type: "Movie", Name: "ความรัก"},
			want: "share",
		},
		{
			name: "mixed thai and latin keeps the latin words",
			item: Item{Type: "Movie", Name: "Thai ไทย"},
			want: "thai",
		},
		{
			name: "empty title falls back to share",
			item: Item{Type: "Movie", Name: "   "},
			want: "share",
		},
		{
			name: "long title is capped",
			item: Item{Type: "Movie", Name: strings.Repeat("a", 80)},
			want: strings.Repeat("a", maxSlug),
		},
		{
			name: "cap does not leave a trailing hyphen",
			item: Item{Type: "Movie", Name: strings.Repeat("a", 39) + " b"},
			want: strings.Repeat("a", 39),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Build(tt.item); got != tt.want {
				t.Fatalf("Build(%+v) = %q, want %q", tt.item, got, tt.want)
			}
		})
	}
}

func TestBuildNeverExceedsColumn(t *testing.T) {
	// 40 slug + hyphen + 12 suffix = 53, inside VARCHAR(64).
	got := Build(Item{Type: "Movie", Name: strings.Repeat("x", 200)})
	if len(got) > maxSlug {
		t.Fatalf("slug length %d exceeds %d", len(got), maxSlug)
	}
}
