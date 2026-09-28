package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"alfred/internal/config"
	"alfred/internal/geoffrey"
	"alfred/internal/plex"
	"alfred/internal/tmdb"
	"alfred/internal/winston"
)

type Server struct {
	mu           sync.RWMutex
	cfg          config.Config
	settings     *config.SettingsStore
	winstonApp   *winston.App
	geoffreyApp  *geoffrey.App
	plexClient   *plex.Client
	tmdbClient   *tmdb.Client
	webFS        fs.FS
}

type LibraryDTO struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type CollectionDTO struct {
	RatingKey  string `json:"ratingKey"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	ChildCount int    `json:"childCount"`
	Temporary  bool   `json:"temporary"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
	ThumbURL   string `json:"thumbUrl,omitempty"`
	ArtURL     string `json:"artUrl,omitempty"`
}

type RecipeDTO struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	PromptAliases      []string `json:"promptAliases"`
	InclusionRules     []string `json:"inclusionRules"`
	ExclusionRules     []string `json:"exclusionRules"`
	OrderingRules      []string `json:"orderingRules"`
	TemporaryByDefault bool     `json:"temporaryByDefault"`
}

type CreateCollectionRequest struct {
	LibraryKey   string   `json:"libraryKey"`
	Name         string   `json:"name"`
	Titles       []string `json:"titles"`
	SourcePrompt string   `json:"sourcePrompt"`
	Temporary    bool     `json:"temporary"`
	ExpiresAt    string   `json:"expiresAt"`
	PosterURL    string   `json:"posterUrl"`
	PosterBase64 string   `json:"posterBase64"`
}

type ApplyCorrectionRequest struct {
	TMDBID               *int   `json:"tmdb_id,omitempty"`
	RelativePathOverride string `json:"relative_path_override,omitempty"`
}

type ReviewListItem struct {
	SourceNZBPath string                   `json:"source_nzb_path"`
	State         winston.ItemState        `json:"state"`
	Confidence    winston.MatchConfidence  `json:"confidence"`
	Metadata      winston.ItemMetadata     `json:"metadata"`
	ProposedPath  string                   `json:"proposed_path"`
	Reason        string                   `json:"reason"`
	Candidates    []winston.CandidateMatch `json:"candidates,omitempty"`
	QueueID       int                      `json:"queue_id,omitempty"`
	Status        string                   `json:"status,omitempty"`
}

func New(
	cfg config.Config,
	settings *config.SettingsStore,
	winstonApp *winston.App,
	geoffreyApp *geoffrey.App,
	plexClient *plex.Client,
	tmdbClient *tmdb.Client,
	webFS fs.FS,
) *Server {
	return &Server{
		cfg:         cfg,
		settings:    settings,
		winstonApp:  winstonApp,
		geoffreyApp: geoffreyApp,
		plexClient:  plexClient,
		tmdbClient:  tmdbClient,
		webFS:       webFS,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health & System Status
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/status", s.handleStatus)

	// Settings
	mux.HandleFunc("/api/settings", s.handleSettings)

	// Winston Endpoints (Namespaced & Aliased)
	registerDual(mux, "/api/winston/review/items", "/api/review/items", s.handleReviewItems)
	registerDual(mux, "/api/winston/review/item", "/api/review/item", s.handleReviewItem)
	registerDual(mux, "/api/winston/review/correct", "/api/review/correct", s.handleReviewCorrect)
	registerDual(mux, "/api/winston/review/approve", "/api/review/approve", s.handleReviewApprove)
	registerDual(mux, "/api/winston/review/import", "/api/review/import", s.handleReviewImport)
	registerDual(mux, "/api/winston/review/reset", "/api/review/reset", s.handleReviewReset)
	registerDual(mux, "/api/winston/review/rescan", "/api/review/rescan", s.handleReviewRescan)
	registerDual(mux, "/api/winston/filebot/status", "/api/filebot/status", s.handleFileBotStatus)

	// Geoffrey Endpoints (Namespaced & Aliased)
	registerDual(mux, "/api/geoffrey/libraries", "/api/libraries", s.handleLibraries)
	registerDual(mux, "/api/geoffrey/collections", "/api/collections", s.handleCollections)
	registerDual(mux, "/api/geoffrey/search", "/api/search", s.handleSearch)
	registerDual(mux, "/api/geoffrey/ideas", "/api/ideas", s.handleIdeas)
	registerDual(mux, "/api/geoffrey/recipes", "/api/recipes", s.handleRecipes)
	registerDual(mux, "/api/geoffrey/poster/upload", "/api/poster/upload", s.handlePosterUpload)
	registerDual(mux, "/api/geoffrey/plex/image", "/api/plex/image", s.handlePlexImage)

	// Delete collection prefix
	mux.HandleFunc("/api/geoffrey/collections/", s.handleCollectionDelete)
	mux.HandleFunc("/api/collections/", s.handleCollectionDelete)

	return withCORS(s.wrapWeb(mux))
}

func registerDual(mux *http.ServeMux, pathA, pathB string, h http.HandlerFunc) {
	mux.HandleFunc(pathA, h)
	if pathB != "" && pathB != pathA {
		mux.HandleFunc(pathB, h)
	}
}

func (s *Server) wrapWeb(apiMux *http.ServeMux) http.Handler {
	var fileServer http.Handler
	if s.webFS != nil {
		fileServer = http.FileServer(http.FS(s.webFS))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			apiMux.ServeHTTP(w, r)
			return
		}
		if s.webFS == nil {
			http.NotFound(w, r)
			return
		}
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "." || p == "" {
			http.ServeFileFS(w, r, s.webFS, "index.html")
			return
		}
		if _, err := fs.Stat(s.webFS, p); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFileFS(w, r, s.webFS, "index.html")
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Plex-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Status & Settings Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"alfred":   "ready",
		"winston":  "ready",
		"geoffrey": "ready",
		"time":     time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plexConnected := s.plexClient != nil && s.plexClient.Configured()
	winstonConfigured := s.cfg.SourceRoot != ""
	altmountConfigured := s.cfg.AltMountBaseURL != ""

	filebotStatus := s.winstonApp.FileBot().Status(context.Background())

	reviewCount := 0
	if s.winstonApp.State() != nil {
		for _, rec := range s.winstonApp.State().Data.Imported {
			if rec.State == winston.StateNeedsReview || rec.State == winston.StateDetected {
				reviewCount++
			}
		}
	}

	collectionCount := 0
	if s.geoffreyApp != nil && s.geoffreyApp.Memory() != nil {
		collectionCount = len(s.geoffreyApp.Memory().Data.History)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"plex_connected":      plexConnected,
		"winston_configured":  winstonConfigured,
		"altmount_configured": altmountConfigured,
		"filebot":             filebotStatus,
		"pending_reviews":     reviewCount,
		"active_collections":  collectionCount,
	})
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.settings.Get())
	case http.MethodPost:
		var incoming config.SettingsDTO
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			writeError(w, http.StatusBadRequest, "invalid settings payload: "+err.Error())
			return
		}
		if err := s.settings.Put(incoming); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save settings: "+err.Error())
			return
		}

		// Update runtime configs
		newCfg := s.settings.ApplyToConfig(s.cfg)
		s.cfg = newCfg

		newPlex := plex.New(newCfg.PlexBaseURL, newCfg.PlexToken, newCfg.PlexPathFrom, newCfg.PlexPathTo)
		newTMDB := tmdb.New(newCfg.TMDBAPIKey)

		s.plexClient = newPlex
		s.tmdbClient = newTMDB
		s.winstonApp.UpdateConfig(newCfg, newPlex)
		s.geoffreyApp.UpdateConfig(newCfg, newPlex, newTMDB)

		log.Printf("alfred: settings updated and reloaded in runtime")
		writeJSON(w, http.StatusOK, s.settings.Get())
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- Winston Handlers ---

func (s *Server) handleReviewItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.sendReviewItems(w)
}

func (s *Server) sendReviewItems(w http.ResponseWriter) {
	state := s.winstonApp.State()
	if state == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		return
	}
	items := make([]ReviewListItem, 0, len(state.Data.Imported))
	for source, rec := range state.Data.Imported {
		reason := ""
		var candidates []winston.CandidateMatch
		if rec.Preview != nil {
			reason = rec.Preview.Reason
			candidates = rec.Preview.Candidates
		}
		items = append(items, ReviewListItem{
			SourceNZBPath: source,
			State:         rec.State,
			Confidence:    rec.Confidence,
			Metadata:      rec.Metadata,
			ProposedPath:  rec.RelativePath,
			Reason:        reason,
			Candidates:    candidates,
			QueueID:       rec.QueueID,
			Status:        rec.Status,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].SourceNZBPath < items[j].SourceNZBPath
	})
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleReviewItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	source := r.URL.Query().Get("source")
	if source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}
	state := s.winstonApp.State()
	if state == nil {
		writeError(w, http.StatusNotFound, "state not available")
		return
	}
	rec, ok := state.Data.Imported[source]
	if !ok {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	reason := ""
	var candidates []winston.CandidateMatch
	if rec.Preview != nil {
		reason = rec.Preview.Reason
		candidates = rec.Preview.Candidates
	}
	writeJSON(w, http.StatusOK, ReviewListItem{
		SourceNZBPath: source,
		State:         rec.State,
		Confidence:    rec.Confidence,
		Metadata:      rec.Metadata,
		ProposedPath:  rec.RelativePath,
		Reason:        reason,
		Candidates:    candidates,
		QueueID:       rec.QueueID,
		Status:        rec.Status,
	})
}

func (s *Server) handleReviewCorrect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	source := r.URL.Query().Get("source")
	if source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}
	var req ApplyCorrectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	proc := s.winstonApp.Processor()
	if proc == nil {
		writeError(w, http.StatusInternalServerError, "processor unavailable")
		return
	}
	if req.TMDBID != nil {
		preview, err := proc.ApplyTMDBCorrection(source, *req.TMDBID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, preview)
		return
	}
	if req.RelativePathOverride != "" {
		preview, err := proc.ApplyRelativePathOverride(source, req.RelativePathOverride)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, preview)
		return
	}
	writeError(w, http.StatusBadRequest, "tmdb_id or relative_path_override is required")
}

func (s *Server) handleReviewApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	source := r.URL.Query().Get("source")
	if source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}
	proc := s.winstonApp.Processor()
	if proc == nil {
		writeError(w, http.StatusInternalServerError, "processor unavailable")
		return
	}
	preview, err := proc.Approve(source)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleReviewImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	source := r.URL.Query().Get("source")
	if source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}
	proc := s.winstonApp.Processor()
	if proc == nil {
		writeError(w, http.StatusInternalServerError, "processor unavailable")
		return
	}
	if err := proc.ImportOne(r.Context(), source); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleReviewReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	source := r.URL.Query().Get("source")
	if source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}
	state := s.winstonApp.State()
	if state == nil {
		writeError(w, http.StatusInternalServerError, "state store unavailable")
		return
	}
	if err := state.Delete(source); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleReviewRescan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	runner := s.winstonApp.QueueRunner()
	proc := s.winstonApp.Processor()
	state := s.winstonApp.State()
	if runner == nil || proc == nil || state == nil {
		writeError(w, http.StatusInternalServerError, "processor unavailable")
		return
	}
	nzbs, err := runner.ListNZBs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, nzbPath := range nzbs {
		if rec, ok := state.Data.Imported[nzbPath]; ok && rec.Status == "completed" {
			continue
		}
		preview := proc.BuildPreview(nzbPath, winston.ItemMetadata{})
		state.Data.Imported[nzbPath] = winston.ImportedRecord{
			RelativePath: preview.ProposedPath,
			Status:       "detected",
			State:        preview.State,
			Confidence:   preview.Confidence,
			Metadata:     preview.Metadata,
			Preview:      preview,
		}
	}
	_ = state.Save()
	s.sendReviewItems(w)
}

func (s *Server) handleFileBotStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, s.winstonApp.FileBot().Status(r.Context()))
}

// --- Geoffrey Handlers ---

func (s *Server) handleLibraries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	libs, err := s.geoffreyApp.Libraries()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	items := make([]LibraryDTO, 0, len(libs))
	for _, lib := range libs {
		items = append(items, LibraryDTO{Key: lib.Key, Title: lib.Title, Type: lib.Type})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Title < items[j].Title })
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		libraryKey := strings.TrimSpace(r.URL.Query().Get("library"))
		if libraryKey == "" {
			writeError(w, http.StatusBadRequest, "library query is required")
			return
		}
		collections, err := s.geoffreyApp.Collections(libraryKey)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		history := s.geoffreyApp.Memory().Data.History
		metaByName := map[string]struct {
			temp    bool
			expires string
		}{}
		for _, item := range history {
			metaByName[strings.ToLower(item.Library+"::"+item.Name)] = struct {
				temp    bool
				expires string
			}{temp: item.Temporary, expires: item.ExpiresAt}
		}
		items := make([]CollectionDTO, 0, len(collections))
		for _, item := range collections {
			meta := metaByName[strings.ToLower(libraryKey+"::"+item.Title)]
			items = append(items, CollectionDTO{
				RatingKey:  item.RatingKey,
				Title:      item.Title,
				Type:       item.Type,
				ChildCount: item.ChildCount,
				Temporary:  meta.temp,
				ExpiresAt:  meta.expires,
				ThumbURL:   proxyImageURL(item.Thumb),
				ArtURL:     proxyImageURL(item.Art),
			})
		}
		sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title) })
		writeJSON(w, http.StatusOK, map[string]any{"items": items})

	case http.MethodPost:
		var req CreateCollectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.LibraryKey = strings.TrimSpace(req.LibraryKey)
		if req.Name == "" || req.LibraryKey == "" {
			writeError(w, http.StatusBadRequest, "name and libraryKey are required")
			return
		}
		if err := s.geoffreyApp.CreateCollectionFromTitles(req.LibraryKey, req.Name, req.Titles, req.SourcePrompt, req.Temporary, req.ExpiresAt); err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		if req.PosterURL != "" || req.PosterBase64 != "" {
			if err := s.geoffreyApp.ApplyCollectionPoster(req.LibraryKey, req.Name, req.PosterURL, req.PosterBase64); err != nil {
				log.Printf("alfred: poster apply warning for %s: %v", req.Name, err)
			}
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleCollectionDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	p := r.URL.Path
	p = strings.TrimPrefix(p, "/api/geoffrey/collections/")
	p = strings.TrimPrefix(p, "/api/collections/")
	parts := strings.Split(p, "/")
	if len(parts) != 2 {
		writeError(w, http.StatusBadRequest, "expected /api/collections/{libraryKey}/{name}")
		return
	}
	libraryKey := parts[0]
	name, err := url.PathUnescape(parts[1])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid collection name")
		return
	}
	if err := s.geoffreyApp.DeleteCollectionByName(libraryKey, name); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	libraryKey := strings.TrimSpace(r.URL.Query().Get("library"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if libraryKey == "" || query == "" {
		writeError(w, http.StatusBadRequest, "library and q are required")
		return
	}
	items, err := s.geoffreyApp.Search(libraryKey, query)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleIdeas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	libraryKey := strings.TrimSpace(r.URL.Query().Get("library"))
	idea := strings.TrimSpace(r.URL.Query().Get("idea"))
	if libraryKey == "" || idea == "" {
		writeError(w, http.StatusBadRequest, "library and idea are required")
		return
	}
	suggestion, err := s.geoffreyApp.SuggestFromIdea(libraryKey, idea)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, suggestion)
}

func (s *Server) handleRecipes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	recipes := s.geoffreyApp.Recipes()
	items := make([]RecipeDTO, 0, len(recipes))
	for _, item := range recipes {
		items = append(items, RecipeDTO{
			ID:                 item.ID,
			Name:               item.Name,
			PromptAliases:      item.PromptAliases,
			InclusionRules:     item.InclusionRules,
			ExclusionRules:     item.ExclusionRules,
			OrderingRules:      item.OrderingRules,
			TemporaryByDefault: item.TemporaryByDefault,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handlePlexImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	imagePath := strings.TrimSpace(r.URL.Query().Get("path"))
	if imagePath == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	resp, err := s.plexClient.FetchImage(imagePath)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handlePosterUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	data, err := fileToDataURL(file, header)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dataUrl": data, "filename": header.Filename})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func proxyImageURL(imagePath string) string {
	if strings.TrimSpace(imagePath) == "" {
		return ""
	}
	return "/api/geoffrey/plex/image?path=" + url.QueryEscape(imagePath)
}

func fileToDataURL(file multipart.File, header *multipart.FileHeader) (string, error) {
	blob, err := io.ReadAll(io.LimitReader(file, 8<<20))
	if err != nil {
		return "", err
	}
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(blob)
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(blob)), nil
}
