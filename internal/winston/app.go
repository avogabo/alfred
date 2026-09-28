package winston

import (
	"context"
	"log"

	"alfred/internal/config"
	"alfred/internal/plex"
)

type App struct {
	cfg             config.Config
	altMount        *AltMountClient
	importProcessor *ImportProcessor
	queueRunner     *QueueRunner
	state           *StateStore
	plex            *plex.Client
}

func New(cfg config.Config, plexClient *plex.Client) (*App, error) {
	alt := NewAltMountClient(cfg)
	state, err := NewStateStore(cfg.SourceRoot)
	if err != nil {
		log.Printf("winston: warning: state store failed in %s: %v", cfg.SourceRoot, err)
	}
	proc := NewImportProcessor(cfg, alt, plexClient, state)
	queue := NewQueueRunner(cfg, proc)
	return &App{
		cfg:             cfg,
		altMount:        alt,
		importProcessor: proc,
		queueRunner:     queue,
		state:           state,
		plex:            plexClient,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	if a.cfg.SourceRoot == "" {
		log.Printf("winston: SOURCE_ROOT not set, file scanning disabled")
		return nil
	}
	if a.cfg.AltMountBaseURL == "" {
		log.Printf("winston: ALTMOUNT_BASE_URL is empty, running in dry bootstrap mode")
		return nil
	}
	log.Printf("winston: background queue runner starting for %s", a.cfg.SourceRoot)
	go func() {
		if err := a.queueRunner.Run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("winston: queue runner ended with error: %v", err)
		}
	}()
	return nil
}

func (a *App) Processor() *ImportProcessor {
	return a.importProcessor
}

func (a *App) QueueRunner() *QueueRunner {
	return a.queueRunner
}

func (a *App) State() *StateStore {
	return a.state
}

func (a *App) FileBot() *FileBotClient {
	if a.importProcessor != nil {
		return a.importProcessor.filebot
	}
	return NewFileBotClient(a.cfg)
}

func (a *App) Config() config.Config {
	return a.cfg
}

func (a *App) UpdateConfig(cfg config.Config, plexClient *plex.Client) {
	a.cfg = cfg
	a.plex = plexClient
	a.altMount = NewAltMountClient(cfg)
	if a.importProcessor != nil {
		a.importProcessor.cfg = cfg
		a.importProcessor.alt = a.altMount
		a.importProcessor.plex = plexClient
		a.importProcessor.filebot = NewFileBotClient(cfg)
	}
	if a.queueRunner != nil {
		a.queueRunner.cfg = cfg
	}
}
