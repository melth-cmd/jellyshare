package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/melth-cmd/jellyshare/internal/config"
	"github.com/melth-cmd/jellyshare/internal/database"
	"github.com/melth-cmd/jellyshare/internal/download"
	"github.com/melth-cmd/jellyshare/internal/handlers"
	"github.com/melth-cmd/jellyshare/internal/jellyfin"
	"github.com/melth-cmd/jellyshare/internal/middleware"
	"github.com/melth-cmd/jellyshare/internal/proxy"
)

func main() {
	cfg := config.Load()

	// Validate required config
	if cfg.JellyfinAPIKey == "" {
		log.Fatal("JELLYSHARE_JELLYFIN_API_KEY is required")
	}
	if cfg.BackendAPIKey == "" {
		log.Fatal("JELLYSHARE_BACKEND_API_KEY is required")
	}
	for _, warning := range cfg.Warnings() {
		log.Printf("WARNING: %s", warning)
	}

	// Initialize database
	db, err := database.New(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations from filesystem
	if err := db.RunMigrationsFromPath("migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Jellyfin client
	jf := jellyfin.NewClient(cfg.JellyfinBaseURL, cfg.JellyfinAPIKey)
	if err := jf.VerifyConnection(context.Background()); err != nil {
		log.Printf("Warning: Could not verify Jellyfin connection: %v", err)
	} else {
		log.Printf("Successfully connected to Jellyfin at %s", cfg.JellyfinBaseURL)
	}

	// Fetch Jellyfin user ID for API calls
	if err := jf.FetchAndSetUserID(context.Background()); err != nil {
		log.Fatalf("Failed to fetch Jellyfin user: %v", err)
	}

	// Initialize session manager
	sessionManager := middleware.NewShareSessionManager(cfg.BackendAPIKey)

	// Initialize handlers
	adminHandler := handlers.NewAdminHandler(db, jf, cfg, sessionManager)
	downloadSigner := download.NewSigner(cfg.BackendAPIKey)
	publicHandler := handlers.NewPublicHandler(db, jf, cfg, sessionManager, downloadSigner)
	streamProxy := proxy.NewStreamProxy(db, jf, cfg, downloadSigner, sessionManager)

	// Create router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger) // chi's logger, with bearer tokens masked
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.BodyLimit(64 << 10))
	r.Use(middleware.ClientIPMiddleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Backend-Key"},
		MaxAge:         300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Admin API (requires backend API key)
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.AdminAuth(cfg.BackendAPIKey))

		r.Post("/shares", adminHandler.CreateShare)
		r.Get("/shares", adminHandler.ListShares)
		r.Post("/shares/bulk/revoke", adminHandler.BulkRevokeShares)
		r.Post("/shares/bulk/delete", adminHandler.BulkDeleteShares)
		r.Get("/shares/{id}", adminHandler.GetShare)
		r.Post("/shares/{id}/revoke", adminHandler.RevokeShare)
		r.Post("/shares/{id}/restore", adminHandler.RestoreShare)
		r.Post("/shares/{id}/duplicate", adminHandler.DuplicateShare)
		r.Patch("/shares/{id}", adminHandler.UpdateShare)
		r.Get("/shares/{id}/analytics", adminHandler.GetShareAnalytics)
		r.Get("/analytics", adminHandler.GetAnalytics)
		r.Get("/stats", adminHandler.GetStats)
		r.Get("/jellyfin/items", adminHandler.SearchLibrary)
		r.Get("/jellyfin/items/{id}/poster", adminHandler.GetItemPoster)
		r.Get("/jellyfin/series/{id}/seasons", adminHandler.ListSeriesSeasons)
		r.Get("/jellyfin/series/{id}/episodes", adminHandler.ListSeriesEpisodes)
		r.Get("/jellyfin/seasons/{id}/episodes", adminHandler.ListSeasonEpisodes)
	})

	// Public API. The calls a viewer makes by hand share a per-IP rate limit.
	// Images and stream segments get their own, ten times looser: one page of
	// episode stills or a few minutes of HLS would otherwise spend the first and
	// lock the viewer out of Play, but they still need some bound.
	r.Route("/api/public", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(cfg.RateLimitRequests, cfg.RateLimitWindow))

			r.Get("/shares/{token}", publicHandler.GetShareInfo)
			// Password guesses get a tighter budget than ordinary page calls:
			// bcrypt is deliberately slow, and this is the one endpoint where a
			// wrong guess is useful to an attacker.
			r.With(httprate.LimitByIP(10, time.Minute)).Post("/shares/{token}/password", publicHandler.ValidatePassword)
			r.Post("/shares/{token}/play", publicHandler.StartPlayback)
			r.Get("/shares/{token}/episodes", publicHandler.GetShareEpisodes)
			r.Get("/shares/{token}/progress", publicHandler.GetShareProgress)
			r.Post("/shares/{token}/episodes/{episodeId}/play", publicHandler.StartEpisodePlayback)
			r.Post("/shares/{token}/download", publicHandler.RequestDownload)
			r.Post("/shares/{token}/episodes/download", publicHandler.RequestAllEpisodesDownload)
			r.Post("/shares/{token}/episodes/{episodeId}/download", publicHandler.RequestEpisodeDownload)
			r.Post("/sessions/{sessionId}/heartbeat", publicHandler.Heartbeat)
			r.Post("/sessions/{sessionId}/finish", publicHandler.FinishPlayback)

			// WebVTT sidecar for text subtitles
			r.Get("/subtitles/{sessionId}/{index}", streamProxy.ServeSubtitle)

			// The file itself, authorised by the ticket the POSTs above hand out
			r.Get("/downloads/{ticket}", streamProxy.ServeDownload)
		})

		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(cfg.RateLimitRequests*10, cfg.RateLimitWindow))

			// Image proxy
			r.Get("/images/{token}/{type}", streamProxy.ServeImage)
			r.Get("/images/{token}/episode/{episodeId}", streamProxy.ServeEpisodeImage)

			// Stream proxy, authorised by its live session
			r.Get("/stream/{sessionId}/*", streamProxy.ServeStream)
		})
	})

	// Serve static frontend
	r.Get("/s/{token}", serveIndex)
	r.Get("/s/{token}/*", serveIndex)

	// Admin UI
	r.Get("/admin", serveIndex)
	r.Get("/admin/*", serveIndex)

	// Serve static assets from filesystem
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/dist/assets"))))

	// The logo set (icons, manifest). Browsers ask for /favicon.ico on their own,
	// whatever the page links, so it is answered from the same folder.
	r.Handle("/brand/*", http.StripPrefix("/brand/", http.FileServer(http.Dir("web/dist/brand"))))
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/dist/brand/favicon.ico")
	})

	// Start background cleanup task
	go startCleanupTask(db, cfg)

	// Create server
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // No timeout for streaming
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on port %d", cfg.Port)
	log.Printf("Public URL: %s", cfg.PublicBaseURL)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/dist/index.html")
}

func startCleanupTask(db *database.DB, cfg *config.Config) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		// Cleanup old sessions
		deleted, err := db.CleanupOldSessions(ctx, 7*24*time.Hour)
		if err != nil {
			log.Printf("Failed to cleanup old sessions: %v", err)
		} else if deleted > 0 {
			log.Printf("Cleaned up %d old sessions", deleted)
		}

		// Remove links that have been dead long enough that nobody needs the row.
		if cfg.ShareRetentionDays > 0 {
			stale, err := db.DeleteStaleShares(ctx, time.Duration(cfg.ShareRetentionDays)*24*time.Hour)
			if err != nil {
				log.Printf("Failed to delete stale shares: %v", err)
			} else if stale > 0 {
				log.Printf("Deleted %d stale shares", stale)
			}
		}
	}
}
