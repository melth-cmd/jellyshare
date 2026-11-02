package config

import (
	"strings"
	"testing"
)

func TestWarningsForUnsafeProductionConfig(t *testing.T) {
	cfg := &Config{
		Environment:     "production",
		BackendAPIKey:   "short",
		PublicBaseURL:   "http://share.example.com",
		DatabaseDSN:     "postgres://jellyshare:jellyshare@postgres:5432/jellyshare?sslmode=disable",
		JellyfinBaseURL: "http://jellyfin:8096",
	}

	warnings := strings.Join(cfg.Warnings(), "\n")
	for _, want := range []string{"shorter than 32 bytes", "not HTTPS", "default jellyshare:jellyshare password"} {
		if !strings.Contains(warnings, want) {
			t.Errorf("warnings = %q, missing %q", warnings, want)
		}
	}
}

func TestWarningsAreQuietForDevelopment(t *testing.T) {
	cfg := &Config{
		Environment:   "development",
		BackendAPIKey: "dev",
		PublicBaseURL: "http://localhost:8098",
		DatabaseDSN:   "postgres://jellyshare:jellyshare@postgres:5432/jellyshare?sslmode=disable",
	}
	if warnings := cfg.Warnings(); len(warnings) != 0 {
		t.Errorf("development warnings = %v, want none", warnings)
	}
}

func TestWarningsAreQuietForSafeProductionConfig(t *testing.T) {
	cfg := &Config{
		Environment:   "production",
		BackendAPIKey: strings.Repeat("a", 64),
		PublicBaseURL: "https://share.example.com",
		DatabaseDSN:   "postgres://jellyshare:a-long-password@postgres:5432/jellyshare?sslmode=disable",
	}
	if warnings := cfg.Warnings(); len(warnings) != 0 {
		t.Errorf("safe production warnings = %v, want none", warnings)
	}
}

func TestUnknownEnvironmentIsTreatedAsProduction(t *testing.T) {
	cfg := &Config{
		Environment:   "staging",
		BackendAPIKey: "short",
		PublicBaseURL: "http://share.example.com",
		DatabaseDSN:   "postgres://jellyshare:secret@postgres:5432/jellyshare",
	}
	if !cfg.IsProduction() {
		t.Error("unknown environment should be treated as production")
	}
	warnings := strings.Join(cfg.Warnings(), "\n")
	if !strings.Contains(warnings, "JELLYSHARE_ENV is not") {
		t.Errorf("warnings = %q, missing environment warning", warnings)
	}
}
