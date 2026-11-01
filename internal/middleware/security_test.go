package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForwardedHeadersOnlyTrustedFromPrivateProxy(t *testing.T) {
	for _, tc := range []struct {
		name       string
		remote     string
		forwarded  string
		wantClient string
	}{
		{
			name:       "direct internet client cannot choose its IP",
			remote:     "203.0.113.9:443",
			forwarded:  "198.51.100.1",
			wantClient: "203.0.113.9",
		},
		{
			name:       "private proxy passes the client it appended",
			remote:     "172.18.0.2:54321",
			forwarded:  "1.2.3.4, 198.51.100.1",
			wantClient: "198.51.100.1",
		},
		{
			name:       "invalid forwarded value falls back to the proxy",
			remote:     "192.168.1.10:1234",
			forwarded:  "not-an-ip",
			wantClient: "192.168.1.10",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			handler := ClientIPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = GetClientIP(r.Context())
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remote
			req.Header.Set("X-Forwarded-For", tc.forwarded)
			handler.ServeHTTP(httptest.NewRecorder(), req)
			if got != tc.wantClient {
				t.Errorf("client IP = %q, want %q", got, tc.wantClient)
			}
		})
	}
}

func TestSecurityHeadersAddHSTSOnlyOverHTTPS(t *testing.T) {
	for _, tc := range []struct {
		name string
		tls  bool
		want bool
	}{
		{name: "plain http", want: false},
		{name: "tls", tls: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.tls {
				req.TLS = &tls.ConnectionState{}
			}
			rec := httptest.NewRecorder()
			SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rec, req)
			if got := rec.Header().Get("Strict-Transport-Security") != ""; got != tc.want {
				t.Errorf("HSTS present = %v, want %v", got, tc.want)
			}
			if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Error("missing nosniff")
			}
		})
	}
}
