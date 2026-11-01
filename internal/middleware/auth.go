package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
)

type contextKey string

const (
	ContextKeyClientIP contextKey = "clientIP"
	ContextKeyIPHash   contextKey = "ipHash"
)

// AdminAuth validates the backend API key for admin endpoints
func AdminAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providedKey := r.Header.Get("X-Backend-Key")
			if providedKey == "" {
				providedKey = r.Header.Get("Authorization")
				providedKey = strings.TrimPrefix(providedKey, "Bearer ")
			}

			// Compare fixed-size digests so the comparison leaks neither the key's
			// contents nor its length.
			providedDigest := sha256.Sum256([]byte(providedKey))
			expectedDigest := sha256.Sum256([]byte(apiKey))
			if subtle.ConstantTimeCompare(providedDigest[:], expectedDigest[:]) != 1 {
				http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Admin responses carry link terms and session details; never let a
			// browser or intermediary keep a copy.
			w.Header().Set("Cache-Control", "no-store")
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIPMiddleware resolves the client address and rewrites RemoteAddr so
// rate limiting, audit logs and the IP context all use the same value.
//
// Forwarding headers are only trusted when the request arrived from a private or
// loopback address - the reverse proxy on the same host or Docker network. A
// client connecting from the internet cannot choose its own rate-limit bucket
// by sending X-Forwarded-For.
func ClientIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := extractClientIP(r)
		ipHash := hashIP(clientIP)

		if ip := net.ParseIP(clientIP); ip != nil {
			r.RemoteAddr = net.JoinHostPort(ip.String(), "0")
		}

		ctx := context.WithValue(r.Context(), ContextKeyClientIP, clientIP)
		ctx = context.WithValue(ctx, ContextKeyIPHash, ipHash)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractClientIP(r *http.Request) string {
	remote := remoteIP(r.RemoteAddr)
	ip := net.ParseIP(remote)
	if ip == nil {
		return remote
	}
	if !ip.IsLoopback() && !ip.IsPrivate() {
		// Direct internet client: forwarding headers are untrusted claims.
		return ip.String()
	}

	// Trust only the value the nearest proxy appended. A proxy such as Caddy
	// appends the address it saw to any X-Forwarded-For the client sent, so the
	// rightmost entry is the one the proxy itself vouches for; earlier entries
	// may be attacker-supplied. CF-Connecting-IP and X-Real-IP are not used
	// because a proxy may pass a client's own copy through unchanged.
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		return ip.String()
	}
	parts := strings.Split(forwarded, ",")
	for i := len(parts) - 1; i >= 0; i -= 1 {
		candidate := strings.TrimSpace(parts[i])
		if parsed := net.ParseIP(candidate); parsed != nil {
			return parsed.String()
		}
	}
	return ip.String()
}

func remoteIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		addr = host
	}
	return strings.Trim(addr, "[]")
}

func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:8]) // First 8 bytes for privacy
}

func GetClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value(ContextKeyClientIP).(string); ok {
		return ip
	}
	return ""
}

func GetIPHash(ctx context.Context) string {
	if hash, ok := ctx.Value(ContextKeyIPHash).(string); ok {
		return hash
	}
	return ""
}
