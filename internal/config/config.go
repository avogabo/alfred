package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	HTTPListenAddr string `json:"http_listen_addr"`

	// Plex Settings (Shared between Winston & Geoffrey)
	PlexBaseURL        string `json:"plex_base_url"`
	PlexToken          string `json:"plex_token"`
	PlexDefaultLibrary string `json:"plex_default_library"`
	PlexPathFrom       string `json:"plex_path_from"`
	PlexPathTo         string `json:"plex_path_to"`

	// Winston Settings (NZB Ingestion, FileBot & AltMount)
	SourceRoot          string        `json:"source_root"`
	AltMountBaseURL     string        `json:"altmount_base_url"`
	AltMountAPIKey      string        `json:"altmount_api_key"`
	AltMountPathFrom    string        `json:"altmount_path_from"`
	AltMountPathTo      string        `json:"altmount_path_to"`
	AltMountStagingDir  string        `json:"altmount_staging_dir"`
	AltMountStagingPath string        `json:"altmount_staging_path"`
	DefaultMode         string        `json:"default_mode"`
	SleepBetweenImports time.Duration `json:"sleep_between_imports"`
	AutoImportMedium    bool          `json:"auto_import_medium"`
	MoviesTemplate      string        `json:"movies_template"`
	SeriesTemplate      string        `json:"series_template"`
	FileBotMovieFormat  string        `json:"filebot_movie_format"`
	FileBotSeriesFormat string        `json:"filebot_series_format"`
	FileBotDB           string        `json:"filebot_db"`
	FileBotBinary       string        `json:"filebot_binary"`
	FileBotHome         string        `json:"filebot_home"`

	// Geoffrey Settings (Plex Collections, TMDB & Automation)
	DataDir          string `json:"data_dir"`
	TMDBAPIKey       string `json:"tmdb_api_key"`
	TelegramBotToken string `json:"telegram_bot_token"`
	TimeZone         string `json:"time_zone"`
	LLMProvider      string `json:"llm_provider"`
	LLMAPIKey        string `json:"llm_api_key"`
	LLMModel         string `json:"llm_model"`
}

type SettingsDTO struct {
	// Plex
	PlexBaseURL        string `json:"plex_base_url"`
	PlexToken          string `json:"plex_token"`
	PlexDefaultLibrary string `json:"plex_default_library"`
	PlexPathFrom       string `json:"plex_path_from"`
	PlexPathTo         string `json:"plex_path_to"`

	// AltMount / Winston
	SourceRoot          string `json:"source_root"`
	AltMountBaseURL     string `json:"altmount_base_url"`
	AltMountAPIKey      string `json:"altmount_api_key"`
	AltMountPathFrom    string `json:"altmount_path_from"`
	AltMountPathTo      string `json:"altmount_path_to"`
	AltMountStagingDir  string `json:"altmount_staging_dir"`
	AltMountStagingPath string `json:"altmount_staging_path"`
	DefaultMode         string `json:"default_mode"`
	SleepBetweenImports string `json:"sleep_between_imports"`
	AutoImportMedium    bool   `json:"auto_import_medium"`
	MoviesTemplate      string `json:"movies_template"`
	SeriesTemplate      string `json:"series_template"`
	FileBotMovieFormat  string `json:"filebot_movie_format"`
	FileBotSeriesFormat string `json:"filebot_series_format"`
	FileBotDB           string `json:"filebot_db"`
	FileBotBinary       string `json:"filebot_binary"`
	FileBotHome         string `json:"filebot_home"`

	// Geoffrey
	DataDir          string `json:"data_dir"`
	TMDBAPIKey       string `json:"tmdb_api_key"`
	TelegramBotToken string `json:"telegram_bot_token"`
	TimeZone         string `json:"time_zone"`
	LLMProvider      string `json:"llm_provider"`
	LLMAPIKey        string `json:"llm_api_key"`
	LLMModel         string `json:"llm_model"`
}

func LoadConfig() Config {
	sleep := 3 * time.Second
	if raw := getFirstEnv("ALFRED_SLEEP_BETWEEN_IMPORTS", "WINSTON_SLEEP_BETWEEN_IMPORTS"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil {
			sleep = d
		}
	}

	mode := getFirstEnvWithFallback("filebot", "ALFRED_DEFAULT_MODE", "WINSTON_DEFAULT_MODE")
	autoImportMedium := getFirstEnvWithFallback("true", "ALFRED_AUTOIMPORT_MEDIUM", "WINSTON_AUTOIMPORT_MEDIUM") != "false"

	dataDir := getFirstEnvWithFallback("/config", "ALFRED_DATA_DIR", "GEOFFREY_DATA_DIR")
	defaultFileBotHome := filepath.Join(dataDir, "filebot")
	filebotHome := getFirstEnvWithFallback(defaultFileBotHome, "FILEBOT_HOME", "ALFRED_FILEBOT_HOME")

	return Config{
		HTTPListenAddr:      getFirstEnvWithFallback(":8091", "ALFRED_HTTP_LISTEN_ADDR", "WINSTON_HTTP_LISTEN_ADDR"),
		PlexBaseURL:         getFirstEnv("ALFRED_PLEX_BASE_URL", "PLEX_BASE_URL", "WINSTON_PLEX_BASE_URL"),
		PlexToken:           getFirstEnv("ALFRED_PLEX_TOKEN", "PLEX_TOKEN", "WINSTON_PLEX_TOKEN"),
		PlexDefaultLibrary: getFirstEnvWithFallback("Películas", "PLEX_DEFAULT_LIBRARY", "ALFRED_PLEX_DEFAULT_LIBRARY"),
		PlexPathFrom:        getFirstEnv("ALFRED_PLEX_PATH_FROM", "WINSTON_PLEX_PATH_FROM"),
		PlexPathTo:          getFirstEnv("ALFRED_PLEX_PATH_TO", "WINSTON_PLEX_PATH_TO"),
		SourceRoot:          getFirstEnv("ALFRED_SOURCE_ROOT", "WINSTON_SOURCE_ROOT"),
		AltMountBaseURL:     getFirstEnv("ALFRED_ALTMOUNT_BASE_URL", "WINSTON_ALTMOUNT_BASE_URL"),
		AltMountAPIKey:      getFirstEnv("ALFRED_ALTMOUNT_API_KEY", "WINSTON_ALTMOUNT_API_KEY"),
		AltMountPathFrom:    getFirstEnv("ALFRED_ALTMOUNT_PATH_FROM", "WINSTON_ALTMOUNT_PATH_FROM"),
		AltMountPathTo:      getFirstEnv("ALFRED_ALTMOUNT_PATH_TO", "WINSTON_ALTMOUNT_PATH_TO"),
		AltMountStagingDir:  getFirstEnv("ALFRED_ALTMOUNT_STAGING_DIR", "WINSTON_ALTMOUNT_STAGING_DIR"),
		AltMountStagingPath: getFirstEnv("ALFRED_ALTMOUNT_STAGING_PATH", "WINSTON_ALTMOUNT_STAGING_PATH"),
		DefaultMode:         mode,
		SleepBetweenImports: sleep,
		AutoImportMedium:    autoImportMedium,
		MoviesTemplate:      getFirstEnvWithFallback("Peliculas/{quality}/{alpha}/{title} ({year})", "ALFRED_MOVIES_TEMPLATE", "WINSTON_MOVIES_TEMPLATE"),
		SeriesTemplate:      getFirstEnvWithFallback("Series/{alpha}/{series}/Temporada {season}/{series} - {episode}", "ALFRED_SERIES_TEMPLATE", "WINSTON_SERIES_TEMPLATE"),
		FileBotMovieFormat:  getFirstEnvWithFallback("Peliculas/{plex}", "ALFRED_FILEBOT_FORMAT_MOVIE", "WINSTON_FILEBOT_FORMAT_MOVIE"),
		FileBotSeriesFormat: getFirstEnvWithFallback("Series/{plex}", "ALFRED_FILEBOT_FORMAT_SERIES", "WINSTON_FILEBOT_FORMAT_SERIES"),
		FileBotDB:           getFirstEnvWithFallback("TheMovieDB", "ALFRED_FILEBOT_DB", "WINSTON_FILEBOT_DB"),
		FileBotBinary:       getFirstEnvWithFallback("/usr/local/bin/filebot", "ALFRED_FILEBOT_BINARY", "WINSTON_FILEBOT_BINARY"),
		FileBotHome:         filebotHome,
		DataDir:             dataDir,
		TMDBAPIKey:          getFirstEnv("TMDB_API_KEY", "ALFRED_TMDB_API_KEY"),
		TelegramBotToken:    getFirstEnv("TELEGRAM_BOT_TOKEN", "ALFRED_TELEGRAM_BOT_TOKEN"),
		TimeZone:            getFirstEnvWithFallback("Europe/Madrid", "TZ", "ALFRED_TIME_ZONE"),
		LLMProvider:         getFirstEnvWithFallback("openai", "LLM_PROVIDER"),
		LLMAPIKey:           getFirstEnv("LLM_API_KEY"),
		LLMModel:            getFirstEnvWithFallback("gpt-4o-mini", "LLM_MODEL"),
	}
}

type SettingsStore struct {
	path string
	mu   sync.Mutex
	data SettingsDTO
}

func NewSettingsStore(configDir string, fallback Config) (*SettingsStore, error) {
	if configDir == "" {
		configDir = "."
	}
	_ = os.MkdirAll(configDir, 0755)
	path := filepath.Join(configDir, ".alfred-settings.json")

	// Fallback to legacy .winston-settings.json if present
	if _, err := os.Stat(path); os.IsNotExist(err) {
		legacyPath := filepath.Join(configDir, ".winston-settings.json")
		if _, lErr := os.Stat(legacyPath); lErr == nil {
			path = legacyPath
		}
	}

	dto := SettingsDTO{
		PlexBaseURL:         fallback.PlexBaseURL,
		PlexToken:           fallback.PlexToken,
		PlexDefaultLibrary: fallback.PlexDefaultLibrary,
		PlexPathFrom:        fallback.PlexPathFrom,
		PlexPathTo:          fallback.PlexPathTo,
		SourceRoot:          fallback.SourceRoot,
		AltMountBaseURL:     fallback.AltMountBaseURL,
		AltMountAPIKey:      fallback.AltMountAPIKey,
		AltMountPathFrom:    fallback.AltMountPathFrom,
		AltMountPathTo:      fallback.AltMountPathTo,
		AltMountStagingDir:  fallback.AltMountStagingDir,
		AltMountStagingPath: fallback.AltMountStagingPath,
		DefaultMode:         fallback.DefaultMode,
		SleepBetweenImports: fallback.SleepBetweenImports.String(),
		AutoImportMedium:    fallback.AutoImportMedium,
		MoviesTemplate:      fallback.MoviesTemplate,
		SeriesTemplate:      fallback.SeriesTemplate,
		FileBotMovieFormat:  fallback.FileBotMovieFormat,
		FileBotSeriesFormat: fallback.FileBotSeriesFormat,
		FileBotDB:           fallback.FileBotDB,
		FileBotBinary:       fallback.FileBotBinary,
		FileBotHome:         fallback.FileBotHome,
		DataDir:             fallback.DataDir,
		TMDBAPIKey:          fallback.TMDBAPIKey,
		TelegramBotToken:    fallback.TelegramBotToken,
		TimeZone:            fallback.TimeZone,
		LLMProvider:         fallback.LLMProvider,
		LLMAPIKey:           fallback.LLMAPIKey,
		LLMModel:            fallback.LLMModel,
	}

	store := &SettingsStore{path: path, data: dto}
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &store.data)
	}

	return store, nil
}

func (s *SettingsStore) Get() SettingsDTO {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data
}

func (s *SettingsStore) Put(v SettingsDTO) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = v
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}

func (s *SettingsStore) ApplyToConfig(cfg Config) Config {
	data := s.Get()
	if data.PlexBaseURL != "" {
		cfg.PlexBaseURL = data.PlexBaseURL
	}
	if data.PlexToken != "" {
		cfg.PlexToken = data.PlexToken
	}
	if data.PlexDefaultLibrary != "" {
		cfg.PlexDefaultLibrary = data.PlexDefaultLibrary
	}
	if data.PlexPathFrom != "" {
		cfg.PlexPathFrom = data.PlexPathFrom
	}
	if data.PlexPathTo != "" {
		cfg.PlexPathTo = data.PlexPathTo
	}
	if data.SourceRoot != "" {
		cfg.SourceRoot = data.SourceRoot
	}
	if data.AltMountBaseURL != "" {
		cfg.AltMountBaseURL = data.AltMountBaseURL
	}
	if data.AltMountAPIKey != "" {
		cfg.AltMountAPIKey = data.AltMountAPIKey
	}
	if data.AltMountPathFrom != "" {
		cfg.AltMountPathFrom = data.AltMountPathFrom
	}
	if data.AltMountPathTo != "" {
		cfg.AltMountPathTo = data.AltMountPathTo
	}
	if data.AltMountStagingDir != "" {
		cfg.AltMountStagingDir = data.AltMountStagingDir
	}
	if data.AltMountStagingPath != "" {
		cfg.AltMountStagingPath = data.AltMountStagingPath
	}
	if data.DefaultMode != "" {
		cfg.DefaultMode = data.DefaultMode
	}
	if data.SleepBetweenImports != "" {
		if d, err := time.ParseDuration(data.SleepBetweenImports); err == nil {
			cfg.SleepBetweenImports = d
		}
	}
	cfg.AutoImportMedium = data.AutoImportMedium
	if data.MoviesTemplate != "" {
		cfg.MoviesTemplate = data.MoviesTemplate
	}
	if data.SeriesTemplate != "" {
		cfg.SeriesTemplate = data.SeriesTemplate
	}
	if data.FileBotMovieFormat != "" {
		cfg.FileBotMovieFormat = data.FileBotMovieFormat
	}
	if data.FileBotSeriesFormat != "" {
		cfg.FileBotSeriesFormat = data.FileBotSeriesFormat
	}
	if data.FileBotDB != "" {
		cfg.FileBotDB = data.FileBotDB
	}
	if data.FileBotBinary != "" {
		cfg.FileBotBinary = data.FileBotBinary
	}
	if data.DataDir != "" {
		cfg.DataDir = data.DataDir
	}
	if data.FileBotHome != "" {
		cfg.FileBotHome = data.FileBotHome
		if cfg.FileBotHome == "/config/filebot" && cfg.DataDir != "" && cfg.DataDir != "/config" {
			cfg.FileBotHome = filepath.Join(cfg.DataDir, "filebot")
		}
	} else if cfg.DataDir != "" {
		cfg.FileBotHome = filepath.Join(cfg.DataDir, "filebot")
	}
	if data.TMDBAPIKey != "" {
		cfg.TMDBAPIKey = data.TMDBAPIKey
	}
	if data.TelegramBotToken != "" {
		cfg.TelegramBotToken = data.TelegramBotToken
	}
	if data.TimeZone != "" {
		cfg.TimeZone = data.TimeZone
	}
	if data.LLMProvider != "" {
		cfg.LLMProvider = data.LLMProvider
	}
	if data.LLMAPIKey != "" {
		cfg.LLMAPIKey = data.LLMAPIKey
	}
	if data.LLMModel != "" {
		cfg.LLMModel = data.LLMModel
	}
	return cfg
}

func getFirstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func getFirstEnvWithFallback(fallback string, keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return fallback
}
