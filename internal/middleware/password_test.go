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
			manager.SetSessionCookie(rec, req, "share-token", "$2a$10$hash")

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

func TestSessionCookieIsBoundToTheCurrentPassword(t *testing.T) {
	manager := NewShareSessionManager("secret")
	req := httptest.NewRequest("GET", "http://example.test/", nil)
	rec := httptest.NewRecorder()
	manager.SetSessionCookie(rec, req, "share-token", "old-hash")

	unlock := rec.Result().Cookies()[0]
	withCookie := httptest.NewRequest("GET", "http://example.test/", nil)
	withCookie.AddCookie(unlock)

	if !manager.GetSessionFromCookie(withCookie, "share-token", "old-hash") {
		t.Error("cookie should unlock the share it was issued for")
	}
	if manager.GetSessionFromCookie(withCookie, "share-token", "new-hash") {
		t.Error("changing the password must invalidate the old unlock cookie")
	}
	if manager.GetSessionFromCookie(withCookie, "other-share", "old-hash") {
		t.Error("cookie must not unlock another share")
	}
}
