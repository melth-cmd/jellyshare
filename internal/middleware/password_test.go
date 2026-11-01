package middleware

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"
)

// A Secure cookie is dropped by the browser on plain HTTP, which made the
// password gate loop forever on a LAN deployment. It must follow the scheme the
// viewer actually used.
func TestSessionCookieSecureFollowsRequest(t *testing.T) {
	manager := NewShareSessionManager("secret")

	for _, tc := range []struct {
		name       string
		forwarded  string
		tls        bool
		wantSecure bool
	}{
		{name: "plain http", wantSecure: false},
		{name: "tls", tls: true, wantSecure: true},
		{name: "forwarded https", forwarded: "https", wantSecure: true},
		{name: "forwarded https with port", forwarded: "https, http", wantSecure: true},
		{name: "forwarded http", forwarded: "http", wantSecure: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.test/", nil)
			if tc.forwarded != "" {
				req.Header.Set("X-Forwarded-Proto", tc.forwarded)
			}
			if tc.tls {
				req.TLS = &tls.ConnectionState{}
			}
			rec := httptest.NewRecorder()
			manager.SetSessionCookie(rec, req, "share-token")

			cookies := rec.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("got %d cookies, want 1", len(cookies))
			}
			if cookies[0].Secure != tc.wantSecure {
				t.Errorf("Secure = %v, want %v", cookies[0].Secure, tc.wantSecure)
			}
			if cookies[0].HttpOnly != true {
				t.Error("cookie must stay HttpOnly")
			}
		})
	}
}
