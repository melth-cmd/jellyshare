package handlers

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 50
)

// Jellyfin item ids are hex, but accept any short path-safe token so this does
// not break if a future Jellyfin version changes the format.
var jellyfinItemIDPattern = regexp.MustCompile(`^[A-Za-z0-9]{1,64}$`)

var adminImageClient = &http.Client{Timeout: 30 * time.Second}

// SearchLibrary backs the admin item picker. An empty q lists recent items.
func (h *AdminHandler) SearchLibrary(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := parseSearchLimit(r.URL.Query().Get("limit"))

	items, err := h.jf.SearchItems(r.Context(), query, limit)
	if err != nil {
		log.Printf("Failed to search Jellyfin library: %v", err)
		writeError(w, http.StatusBadGateway, "failed to search the Jellyfin library")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *AdminHandler) ListSeriesSeasons(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !jellyfinItemIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}
	seasons, err := h.jf.BrowseSeasons(r.Context(), id)
	if err != nil {
		log.Printf("Failed to list seasons for %s: %v", id, err)
		writeError(w, http.StatusBadGateway, "failed to load seasons from Jellyfin")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"seasons": seasons})
}

func (h *AdminHandler) ListSeriesEpisodes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !jellyfinItemIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}
	episodes, err := h.jf.BrowseSeriesEpisodes(r.Context(), id)
	if err != nil {
		log.Printf("Failed to list episodes for series %s: %v", id, err)
		writeError(w, http.StatusBadGateway, "failed to load episodes from Jellyfin")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"episodes": episodes})
}

func (h *AdminHandler) ListSeasonEpisodes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !jellyfinItemIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid season id")
		return
	}
	episodes, err := h.jf.BrowseEpisodes(r.Context(), id)
	if err != nil {
		log.Printf("Failed to list episodes for season %s: %v", id, err)
		writeError(w, http.StatusBadGateway, "failed to load episodes from Jellyfin")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"episodes": episodes})
}

func parseSearchLimit(raw string) int {
	if raw == "" {
		return defaultSearchLimit
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return defaultSearchLimit
	}
	return min(limit, maxSearchLimit)
}

// GetItemPoster proxies a Jellyfin poster so the browser never holds the
// Jellyfin API key. The admin UI fetches it with X-Backend-Key.
func (h *AdminHandler) GetItemPoster(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !jellyfinItemIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	imageURL := h.jf.GetPosterURL(id)
	sizing := url.Values{}
	for _, name := range []string{"maxWidth", "maxHeight"} {
		value := r.URL.Query().Get(name)
		if value == "" {
			continue
		}
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 || n > 2000 {
			continue
		}
		sizing.Set(name, strconv.Itoa(n))
	}
	if encoded := sizing.Encode(); encoded != "" {
		imageURL += "?" + encoded
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, imageURL, nil)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", h.jf.AuthHeader())

	resp, err := adminImageClient.Do(req)
	if err != nil {
		http.Error(w, "error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		w.Header().Set("Content-Length", contentLength)
	}
	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(resp.StatusCode)
		return
	}
	// Admin-only image; do not let a shared cache hold it.
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, resp.Body)
}
