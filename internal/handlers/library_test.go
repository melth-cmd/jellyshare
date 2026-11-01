package handlers

import "testing"

func TestParseSearchLimit(t *testing.T) {
	for raw, want := range map[string]int{
		"":     defaultSearchLimit,
		"0":    defaultSearchLimit,
		"-3":   defaultSearchLimit,
		"nope": defaultSearchLimit,
		"5":    5,
		"500":  maxSearchLimit,
		"50":   50,
	} {
		if got := parseSearchLimit(raw); got != want {
			t.Errorf("parseSearchLimit(%q) = %d, want %d", raw, got, want)
		}
	}
}
