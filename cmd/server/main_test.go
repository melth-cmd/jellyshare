package main

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestWebManifestMIME(t *testing.T) {
	const want = "application/manifest+json"
	if got := mime.TypeByExtension(".webmanifest"); got != want {
		t.Fatalf("mime.TypeByExtension(.webmanifest) = %q, want %q", got, want)
	}
}

func TestFileServerServesWebManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.webmanifest")
	if err := os.WriteFile(path, []byte(`{"name":"test"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/manifest.webmanifest", nil)
	rec := httptest.NewRecorder()
	http.FileServer(http.Dir(dir)).ServeHTTP(rec, req)

	const want = "application/manifest+json"
	if got := rec.Header().Get("Content-Type"); got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
}
