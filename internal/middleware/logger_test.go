package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestLoggerMasksDownloadTickets(t *testing.T) {
	var buf bytes.Buffer
	logged := chimiddleware.RequestLogger(maskingFormatter{
		next: &chimiddleware.DefaultLogFormatter{Logger: log.New(&buf, "", 0), NoColor: true},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The handler still sees the real ticket.
		if !strings.HasSuffix(r.RequestURI, "SECRET-TICKET") {
			t.Errorf("handler got %q", r.RequestURI)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/public/downloads/SECRET-TICKET", nil)
	logged.ServeHTTP(httptest.NewRecorder(), req)

	if strings.Contains(buf.String(), "SECRET-TICKET") {
		t.Fatalf("ticket leaked into the log: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "/api/public/downloads/[ticket]") {
		t.Fatalf("download request not logged: %s", buf.String())
	}
}

func TestLoggerMasksPublicBearerTokens(t *testing.T) {
	for _, tc := range []struct {
		path string
		want string
		leak string
	}{
		{"/api/public/shares/SHARE-SECRET", "/api/public/shares/[share]", "SHARE-SECRET"},
		{"/api/public/shares/SHARE-SECRET/play", "/api/public/shares/[share]/play", "SHARE-SECRET"},
		{"/api/public/images/SHARE-SECRET/poster", "/api/public/images/[share]/poster", "SHARE-SECRET"},
		{"/api/public/stream/SESSION-SECRET/master.m3u8", "/api/public/stream/[session]/master.m3u8", "SESSION-SECRET"},
		{"/api/public/subtitles/SESSION-SECRET/2.vtt", "/api/public/subtitles/[session]/2.vtt", "SESSION-SECRET"},
	} {
		var buf bytes.Buffer
		logged := chimiddleware.RequestLogger(maskingFormatter{
			next: &chimiddleware.DefaultLogFormatter{Logger: log.New(&buf, "", 0), NoColor: true},
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.RequestURI, tc.leak) {
				t.Errorf("handler got %q", r.RequestURI)
			}
		}))

		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		logged.ServeHTTP(httptest.NewRecorder(), req)

		if strings.Contains(buf.String(), tc.leak) {
			t.Errorf("%s leaked into the log: %s", tc.leak, buf.String())
		}
		if !strings.Contains(buf.String(), tc.want) {
			t.Errorf("log = %s, want %s", buf.String(), tc.want)
		}
	}
}
