package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"alfred/internal/config"
	"alfred/internal/geoffrey"
	"alfred/internal/plex"
	"alfred/internal/server"
	"alfred/internal/tmdb"
	"alfred/internal/winston"
)

func main() {
	cfg := config.LoadConfig()

	settingsStore, err := config.NewSettingsStore(cfg.DataDir, cfg)
	if err != nil {
		log.Printf("alfred: warning: settings store init error: %v", err)
	} else {
		cfg = settingsStore.ApplyToConfig(cfg)
	}

	plexClient := plex.New(cfg.PlexBaseURL, cfg.PlexToken, cfg.PlexPathFrom, cfg.PlexPathTo)
	tmdbClient := tmdb.New(cfg.TMDBAPIKey)

	winstonApp, err := winston.New(cfg, plexClient)
	if err != nil {
		log.Fatalf("alfred: failed to initialize winston: %v", err)
	}

	geoffreyApp, err := geoffrey.New(cfg, plexClient, tmdbClient)
	if err != nil {
		log.Fatalf("alfred: failed to initialize geoffrey: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("alfred: media butler starting (port=%s)", cfg.HTTPListenAddr)

	// Start modules
	if err := winstonApp.Start(ctx); err != nil {
		log.Printf("alfred: winston startup warning: %v", err)
	}
	geoffreyApp.Start()

	// Extract web embedded filesystem
	var staticFS fs.FS
	if sub, err := fs.Sub(webFS, "web/dist"); err == nil {
		staticFS = sub
	}

	srv := server.New(cfg, settingsStore, winstonApp, geoffreyApp, plexClient, tmdbClient, staticFS)
	httpServer := &http.Server{
		Addr:         cfg.HTTPListenAddr,
		Handler:      srv.Routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		log.Printf("alfred: web ui and api listening at http://0.0.0.0%s", cfg.HTTPListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("alfred: http server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("alfred: shutting down gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	log.Printf("alfred: goodbye")
}
