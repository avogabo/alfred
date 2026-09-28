package winston

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"alfred/internal/config"
)

type FileBotClient struct {
	cfg config.Config
}

type FileBotResolveResult struct {
	RelativePath string `json:"relative_path"`
	RawOutput    string `json:"raw_output"`
	Method       string `json:"method"`
	EpisodeTitle string `json:"episode_title,omitempty"`
	TVDBID       int    `json:"tvdb_id,omitempty"`
}

type FileBotStatus struct {
	Enabled        bool   `json:"enabled"`
	Available      bool   `json:"available"`
	Mode           string `json:"mode"`
	Binary         string `json:"binary"`
	Home           string `json:"home"`
	DB             string `json:"db"`
	LicensePresent bool   `json:"license_present"`
}

func NewFileBotClient(cfg config.Config) *FileBotClient {
	return &FileBotClient{cfg: cfg}
}

func (f *FileBotClient) Enabled() bool {
	return true
}

func (f *FileBotClient) Available(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, f.cfg.FileBotBinary, "-version")
	return cmd.Run() == nil
}

func (f *FileBotClient) Status(ctx context.Context) FileBotStatus {
	home := strings.TrimSpace(f.cfg.FileBotHome)
	licensePresent := false
	if home != "" {
		if entries, err := os.ReadDir(home); err == nil {
			for _, entry := range entries {
				name := strings.ToLower(entry.Name())
				if strings.Contains(name, "license") || strings.HasSuffix(name, ".psm") {
					licensePresent = true
					break
				}
			}
		}
	}
	return FileBotStatus{
		Enabled:        f.Enabled(),
		Available:      f.Available(ctx),
		Mode:           strings.TrimSpace(f.cfg.DefaultMode),
		Binary:         strings.TrimSpace(f.cfg.FileBotBinary),
		Home:           home,
		DB:             strings.TrimSpace(f.cfg.FileBotDB),
		LicensePresent: licensePresent,
	}
}

func (f *FileBotClient) Resolve(ctx context.Context, sourceNZB string, meta ItemMetadata) (*FileBotResolveResult, error) {
	if !f.Enabled() {
		return nil, nil
	}
	if strings.TrimSpace(f.cfg.FileBotBinary) == "" {
		return nil, nil
	}
	if res, err := f.resolveWithFileBot(ctx, sourceNZB, meta); err == nil && res != nil && strings.TrimSpace(res.RelativePath) != "" {
		res.RelativePath = applyDetectedMovieQuality(res.RelativePath, meta)
		res.RelativePath = directoryOnlyRelativePath(res.RelativePath)
		return res, nil
	} else if err != nil {
		fb := applyDetectedMovieQualityResult(f.resolveFallback(sourceNZB, meta), meta)
		fb.RelativePath = directoryOnlyRelativePath(fb.RelativePath)
		fb.RawOutput = err.Error()
		return fb, err
	}
	res := applyDetectedMovieQualityResult(f.resolveFallback(sourceNZB, meta), meta)
	res.RelativePath = directoryOnlyRelativePath(res.RelativePath)
	return res, nil
}

func (f *FileBotClient) resolveWithFileBot(ctx context.Context, sourceNZB string, meta ItemMetadata) (*FileBotResolveResult, error) {
	format := f.cfg.FileBotMovieFormat
	if normalizeKind(meta.Kind) == "series" {
		format = f.cfg.FileBotSeriesFormat
	}
	format = strings.TrimSpace(strings.ReplaceAll(format, "\r", ""))
	format = strings.TrimSpace(format)
	if format == "" {
		return nil, nil
	}

	tmpDir, err := os.MkdirTemp("", "winston-filebot-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	probe := filepath.Join(tmpDir, buildFileBotProbeName(sourceNZB, meta))
	if err := os.WriteFile(probe, []byte{}, 0644); err != nil {
		return nil, err
	}

	args := []string{
		"-rename",
		probe,
		"--db", chooseFileBotDB(meta, f.cfg.FileBotDB),
		"--lang", "es",
		"--format", format,
		"--action", "test",
		"--output", tmpDir,
		"--conflict", "override",
		"-non-strict",
	}
	if meta.TMDBID > 0 {
		args = append(args, "--q", fmt.Sprintf("tmdbid=%d", meta.TMDBID))
	} else if meta.IMDBID != "" {
		args = append(args, "--q", meta.IMDBID)
	} else if meta.Title != "" {
		q := meta.Title
		if meta.Year > 0 {
			q = fmt.Sprintf("%s %d", q, meta.Year)
		}
		args = append(args, "--q", q)
	}
	if normalizeKind(meta.Kind) == "series" && meta.Season > 0 {
		args = append(args, "--filter", fmt.Sprintf("s == %d", meta.Season))
	}

	cmd := exec.CommandContext(ctx, f.cfg.FileBotBinary, args...)
	cmd.Env = append(os.Environ(), "FILEBOT_HOME="+f.cfg.FileBotHome)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if rel, ok := parseFileBotOutput(stdout.String(), tmpDir); ok && strings.TrimSpace(rel) != "" {
			return &FileBotResolveResult{RelativePath: filepath.ToSlash(rel), RawOutput: stdout.String(), Method: "filebot", EpisodeTitle: detectEpisodeTitleFromPath(rel), TVDBID: detectTVDBIDFromPath(rel)}, nil
		}
		return nil, fmt.Errorf("filebot failed: %w stderr=%s stdout=%s", err, strings.TrimSpace(stderr.String()), strings.TrimSpace(stdout.String()))
	}

	rel, ok := parseFileBotOutput(stdout.String(), tmpDir)
	if !ok || strings.TrimSpace(rel) == "" {
		return nil, fmt.Errorf("filebot output did not contain target path")
	}
	return &FileBotResolveResult{RelativePath: filepath.ToSlash(rel), RawOutput: stdout.String(), Method: "filebot", EpisodeTitle: detectEpisodeTitleFromPath(rel), TVDBID: detectTVDBIDFromPath(rel)}, nil
}

func parseFileBotOutput(out, root string) (string, bool) {
	root = filepath.Clean(root)
	reTest := regexp.MustCompile(`\[TEST\]\s+from\s+\[(.*?)\]\s+to\s+\[(.*?)\]`)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := reTest.FindStringSubmatch(line); len(m) == 3 {
			candidate := strings.TrimSpace(m[2])
			if rel, ok := trimToRelative(candidate, root); ok {
				return rel, true
			}
		}
		if strings.Contains(line, "=>") {
			parts := strings.Split(line, "=>")
			candidate := strings.TrimSpace(parts[len(parts)-1])
			candidate = strings.Trim(candidate, "'")
			if rel, ok := trimToRelative(candidate, root); ok {
				return rel, true
			}
		}
		if rel, ok := trimToRelative(line, root); ok {
			return rel, true
		}
	}
	return "", false
}

func trimToRelative(candidate, root string) (string, bool) {
	candidate = filepath.Clean(candidate)
	if candidate == root {
		return "", false
	}
	if !strings.HasPrefix(candidate, root+string(filepath.Separator)) {
		return "", false
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == "." {
		return "", false
	}
	return rel, true
}

func chooseFileBotDB(meta ItemMetadata, fallback string) string {
	if meta.TVDBID > 0 {
		return "TheTVDB"
	}
	if normalizeKind(meta.Kind) == "series" {
		return "TheTVDB"
	}
	if normalizeKind(meta.Kind) == "movie" {
		return "TheMovieDB"
	}
	if strings.TrimSpace(fallback) == "" {
		return "TheMovieDB"
	}
	return fallback
}

func buildFileBotProbeName(sourceNZB string, meta ItemMetadata) string {
	base := cleanupTitle(strings.TrimSuffix(filepath.Base(sourceNZB), filepath.Ext(sourceNZB)))
	if strings.TrimSpace(meta.Title) != "" {
		base = strings.TrimSpace(meta.Title)
		if meta.Year > 0 {
			base = fmt.Sprintf("%s (%d)", base, meta.Year)
		}
		if normalizeKind(meta.Kind) == "series" {
			if meta.Season > 0 && meta.Episode > 0 {
				base = fmt.Sprintf("%s %02dx%02d", base, meta.Season, meta.Episode)
			} else if meta.Season > 0 {
				base = fmt.Sprintf("%s %02dx01", base, meta.Season)
			}
		}
	}
	return sanitizeProbeFilename(base) + ".mkv"
}

func sanitizeProbeFilename(name string) string {
	replacer := strings.NewReplacer("/", " ", "\\", " ", ":", " ", "*", " ", "?", " ", "\"", " ", "<", " ", ">", " ", "|", " ")
	name = replacer.Replace(name)
	name = strings.Join(strings.Fields(name), " ")
	if strings.TrimSpace(name) == "" {
		return "probe"
	}
	return name
}

func (f *FileBotClient) resolveFallback(sourceNZB string, meta ItemMetadata) *FileBotResolveResult {
	title := strings.TrimSpace(meta.Title)
	if title == "" {
		title = cleanupTitle(strings.TrimSuffix(filepath.Base(sourceNZB), filepath.Ext(sourceNZB)))
	}
	kind := normalizeKind(meta.Kind)
	quality := strings.TrimSpace(meta.Quality)
	if quality == "" {
		quality = detectQuality(sourceNZB)
	}
	alpha := "#"
	if title != "" {
		r := []rune(strings.ToUpper(title))
		if len(r) > 0 && regexp.MustCompile(`[A-Z0-9ÁÉÍÓÚÑ]`).MatchString(string(r[0])) {
			alpha = string(r[0])
		}
	}
	movieFmt := f.cfg.FileBotMovieFormat
	seriesFmt := f.cfg.FileBotSeriesFormat
	if strings.TrimSpace(movieFmt) == "" {
		movieFmt = "Peliculas/{quality}/{alpha}/{title} ({year})"
	}
	if strings.TrimSpace(seriesFmt) == "" {
		seriesFmt = "Series/{alpha}/{series}/Temporada {season}/{series} - {episode}"
	}
	if strings.Contains(seriesFmt, "id+") || strings.Contains(seriesFmt, "episode.special") || strings.Contains(seriesFmt, "{\"") {
		seriesFmt = "Series/{alpha}/{series} ({year}) {tvdb}/Temporada {season}/{series} ({year}) - {episode}{episode_title_suffix}"
	}
	if strings.Contains(movieFmt, "id+") || strings.Contains(movieFmt, "{\"") {
		movieFmt = "Peliculas/{quality}/{alpha}/{title} ({year})/{title} ({year})"
	}
	mapping := map[string]string{
		"title":   title,
		"series":  title,
		"year":    maybeInt(meta.Year),
		"season":  twoDigits(defaultInt(meta.Season, 1)),
		"episode": episodeToken(meta.Season, meta.Episode),
		"episode_title": strings.TrimSpace(meta.ResolvedEpisodeTitle),
		"episode_title_suffix": episodeTitleSuffix(meta.ResolvedEpisodeTitle),
		"quality": quality,
		"vf":      quality,
		"alpha":   alpha,
		"plex":    title,
		"tvdb":    tvdbToken(meta),
	}
	format := movieFmt
	if kind == "series" {
		format = seriesFmt
	}
	resolved := applyTokenFormat(format, mapping)
	return &FileBotResolveResult{RelativePath: filepath.ToSlash(strings.Trim(resolved, "/ ")), Method: "fallback"}
}

func applyTokenFormat(format string, mapping map[string]string) string {
	out := format
	for k, v := range mapping {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	out = regexp.MustCompile(`/+`).ReplaceAllString(out, "/")
	return out
}

func applyDetectedMovieQualityResult(res *FileBotResolveResult, meta ItemMetadata) *FileBotResolveResult {
	if res == nil {
		return nil
	}
	res.RelativePath = applyDetectedMovieQuality(res.RelativePath, meta)
	return res
}

func applyDetectedMovieQuality(rel string, meta ItemMetadata) string {
	if normalizeKind(meta.Kind) != "movie" {
		return rel
	}
	q := strings.TrimSpace(meta.Quality)
	if q == "" || q == "unknown" {
		return rel
	}
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return rel
	}
	parts := strings.Split(rel, "/")
	if len(parts) == 0 {
		return rel
	}
	if parts[0] == "Peliculas" {
		if len(parts) > 1 && parts[1] == q {
			return rel
		}
		return filepath.ToSlash(filepath.Join("Peliculas", q, strings.Join(parts[1:], "/")))
	}
	return filepath.ToSlash(filepath.Join("Peliculas", q, rel))
}

func detectEpisodeTitleFromPath(rel string) string {
	base := filepath.Base(rel)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if idx := strings.LastIndex(base, " - "); idx >= 0 && idx+3 < len(base) {
		return strings.TrimSpace(base[idx+3:])
	}
	return ""
}

func detectTVDBIDFromPath(rel string) int {
	re := regexp.MustCompile(`\{tvdb-(\d+)\}`)
	m := re.FindStringSubmatch(rel)
	if len(m) != 2 {
		return 0
	}
	v, _ := strconv.Atoi(m[1])
	return v
}

func directoryOnlyRelativePath(rel string) string {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return rel
	}
	parts := strings.Split(rel, "/")
	if len(parts) <= 1 {
		return rel
	}
	last := parts[len(parts)-1]
	if strings.Contains(last, ".") {
		return strings.Join(parts[:len(parts)-1], "/")
	}
	return rel
}

func detectQuality(source string) string {
	low := strings.ToLower(source)
	switch {
	case strings.Contains(low, "2160") || strings.Contains(low, "4k"):
		return "2160p"
	case strings.Contains(low, "1080"):
		return "1080p"
	case strings.Contains(low, "720"):
		return "720p"
	default:
		return "unknown"
	}
}

func maybeInt(v int) string {
	if v <= 0 {
		return ""
	}
	return strconv.Itoa(v)
}

func defaultInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

func twoDigits(v int) string {
	return fmt.Sprintf("%02d", v)
}

func episodeToken(season, episode int) string {
	return fmt.Sprintf("%02dx%02d", defaultInt(season, 1), defaultInt(episode, 1))
}

func tvdbToken(meta ItemMetadata) string {
	if meta.TVDBID > 0 {
		return fmt.Sprintf("{tvdb-%d}", meta.TVDBID)
	}
	return "{tvdb}"
}

func episodeTitleSuffix(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	return " - " + title
}
