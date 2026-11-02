package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment             string
	Port                    int
	DatabaseDSN             string
	JellyfinBaseURL         string
	JellyfinAPIKey          string
	BackendAPIKey           string
	PublicBaseURL           string
	SessionHeartbeatTimeout time.Duration
	LogLevel                string
	RateLimitRequests       int
	RateLimitWindow         time.Duration
	// MaxTranscodeBitrate caps the target bitrate handed to Jellyfin for a
	// transcode. It never applies to direct stream, which is untouched.
	MaxTranscodeBitrate int
	// StreamVideoCodec / StreamAudioCodec tell Jellyfin what the share page can
	// actually play. Naming a codec does not force a transcode: a source that
	// already matches is stream-copied. Set either to an empty string to send no
	// preference, which is what the backend did before and what leaves AV1 or HEVC
	// to reach the browser untouched.
	StreamVideoCodec string
	StreamAudioCodec string
	// AllowDownloads lets viewers save the original file - and, for a season or
	// series, every episode as one ZIP. A download is charged as one play.
	AllowDownloads bool
	// ShareRetentionDays deletes links this many days after they expired or were
	// revoked. Zero keeps them forever.
	ShareRetentionDays int
}

func Load() *Config {
	return &Config{
		Environment:             getEnv("JELLYSHARE_ENV", "production"),
		Port:                    getEnvInt("JELLYSHARE_PORT", 8097),
		DatabaseDSN:             getEnv("JELLYSHARE_DB_DSN", "postgres://jellyshare:jellyshare@localhost:5432/jellyshare?sslmode=disable"),
		JellyfinBaseURL:         getEnv("JELLYSHARE_JELLYFIN_BASE_URL", "http://localhost:8096"),
		JellyfinAPIKey:          getEnv("JELLYSHARE_JELLYFIN_API_KEY", ""),
		BackendAPIKey:           getEnv("JELLYSHARE_BACKEND_API_KEY", ""),
		PublicBaseURL:           getEnv("JELLYSHARE_PUBLIC_BASE_URL", "http://localhost:8097"),
		SessionHeartbeatTimeout: time.Duration(getEnvInt("JELLYSHARE_SESSION_HEARTBEAT_TIMEOUT_SECONDS", 120)) * time.Second,
		LogLevel:                getEnv("JELLYSHARE_LOG_LEVEL", "info"),
		RateLimitRequests:       getEnvInt("JELLYSHARE_RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:         time.Duration(getEnvInt("JELLYSHARE_RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,
		MaxTranscodeBitrate:     getEnvInt("JELLYSHARE_MAX_TRANSCODE_BITRATE", 20000000),
		StreamVideoCodec:        getEnvAllowEmpty("JELLYSHARE_STREAM_VIDEO_CODEC", "h264"),
		StreamAudioCodec:        getEnvAllowEmpty("JELLYSHARE_STREAM_AUDIO_CODEC", "aac"),
		AllowDownloads:          getEnvBool("JELLYSHARE_ALLOW_DOWNLOADS", true),
		ShareRetentionDays:      getEnvInt("JELLYSHARE_SHARE_RETENTION_DAYS", 365),
	}
}

// IsProduction reports whether startup warnings should be applied. Development
// is the only value that relaxes them; anything else is treated as production.
func (c *Config) IsProduction() bool {
	return !strings.EqualFold(c.Environment, "development")
}

// Warnings returns configuration problems that are not fatal but are unsafe for
// an internet-facing deployment. It never includes a secret value.
func (c *Config) Warnings() []string {
	if !c.IsProduction() {
		return nil
	}

	warnings := []string{}
	if len(c.BackendAPIKey) < 32 {
		warnings = append(warnings, "JELLYSHARE_BACKEND_API_KEY is shorter than 32 bytes; generate one with: openssl rand -hex 32")
	}
	if !strings.HasPrefix(strings.ToLower(c.PublicBaseURL), "https://") {
		warnings = append(warnings, "JELLYSHARE_PUBLIC_BASE_URL is not HTTPS; public links and the admin key will travel in clear text")
	}
	if strings.Contains(c.DatabaseDSN, "jellyshare:jellyshare@") {
		warnings = append(warnings, "the database connection still uses the default jellyshare:jellyshare password")
	}
	if !strings.EqualFold(c.Environment, "production") && !strings.EqualFold(c.Environment, "development") {
		warnings = append(warnings, "JELLYSHARE_ENV is not 'production' or 'development'; treating it as production")
	}
	return warnings
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAllowEmpty differs from getEnv: an explicitly empty value is honoured
// rather than falling back to the default, so a codec preference can be disabled.
func getEnvAllowEmpty(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
