package geoffrey

import (
	"log"

	"alfred/internal/config"
	"alfred/internal/geoffrey/memory"
	"alfred/internal/plex"
	"alfred/internal/tmdb"
)

type App struct {
	cfg    config.Config
	memory *memory.Store
	plex   *plex.Client
	tmdb   *tmdb.Client
}

func New(cfg config.Config, plexClient *plex.Client, tmdbClient *tmdb.Client) (*App, error) {
	store, err := memory.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	return &App{
		cfg:    cfg,
		memory: store,
		plex:   plexClient,
		tmdb:   tmdbClient,
	}, nil
}

func (a *App) Start() {
	if a.memory.Data.UserPreferences.DefaultMovieLibrary == "" {
		a.memory.Data.UserPreferences.DefaultMovieLibrary = a.cfg.PlexDefaultLibrary
	}
	if len(a.memory.Data.Recipes) == 0 {
		a.memory.Data.Recipes = memory.DefaultRecipes()
	}
	_ = a.memory.Save()

	a.StartSchedulers()
	if a.cfg.TelegramBotToken != "" {
		go func() {
			if err := a.RunTelegram(); err != nil {
				log.Printf("geoffrey: telegram error: %v", err)
			}
		}()
	}

	if !a.plex.Configured() {
		log.Printf("geoffrey: plex is not configured yet, waiting for settings")
		return
	}
	libs, err := a.plex.Libraries()
	if err != nil {
		log.Printf("geoffrey: warning: plex libraries check failed: %v", err)
	} else {
		log.Printf("geoffrey: ready, detected %d plex libraries", len(libs))
	}
}

func (a *App) Memory() *memory.Store {
	return a.memory
}

func (a *App) Plex() *plex.Client {
	return a.plex
}

func (a *App) TMDB() *tmdb.Client {
	return a.tmdb
}

func (a *App) Config() config.Config {
	return a.cfg
}

func (a *App) UpdateConfig(cfg config.Config, plexClient *plex.Client, tmdbClient *tmdb.Client) {
	a.cfg = cfg
	a.plex = plexClient
	a.tmdb = tmdbClient
}
