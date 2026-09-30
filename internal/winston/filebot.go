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
	TMDBID       int    `json:"tmdb_id,omitempty"`
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

	candidateDirs := []string{}
	if home != "" {
		candidateDirs = append(candidateDirs, home)
	}
	if f.cfg.DataDir != "" {
		candidateDirs = append(candidateDirs, filepath.Join(f.cfg.DataDir, "filebot"))
	}
	candidateDirs = append(candidateDirs, "/config/filebot")

	for _, dir := range candidateDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				name := strings.ToLower(entry.Name())
				if strings.Contains(name, "license") || strings.HasSuffix(name, ".psm") {
					licensePresent = true
					home = dir
					break
				}
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "data", ".license")); err == nil {
			licensePresent = true
			home = dir
			break
		}
		if licensePresent {
			break
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
	if f.Available(ctx) {
		if res, err := f.resolveWithFileBot(ctx, sourceNZB, meta); err == nil && res != nil && strings.TrimSpace(res.RelativePath) != "" {
			res.RelativePath = applyDetectedMovieQuality(res.RelativePath, meta)
			res.RelativePath = directoryOnlyRelativePath(res.RelativePath)
			return res, nil
		}
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
			return &FileBotResolveResult{
				RelativePath: filepath.ToSlash(rel),
				RawOutput:    stdout.String(),
				Method:       "filebot",
				EpisodeTitle: detectEpisodeTitleFromPath(rel),
				TVDBID:       detectTVDBIDFromPath(rel),
				TMDBID:       detectTMDBIDFromPath(rel),
			}, nil
		}
		return nil, fmt.Errorf("filebot failed: %w stderr=%s stdout=%s", err, strings.TrimSpace(stderr.String()), strings.TrimSpace(stdout.String()))
	}

	rel, ok := parseFileBotOutput(stdout.String(), tmpDir)
	if !ok || strings.TrimSpace(rel) == "" {
		return nil, fmt.Errorf("filebot output did not contain target path")
	}
	return &FileBotResolveResult{
		RelativePath: filepath.ToSlash(rel),
		RawOutput:    stdout.String(),
		Method:       "filebot",
		EpisodeTitle: detectEpisodeTitleFromPath(rel),
		TVDBID:       detectTVDBIDFromPath(rel),
		TMDBID:       detectTMDBIDFromPath(rel),
	}, nil
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
		movieFmt = `Peliculas/{vf}/{az}/{n} ({y}) {"{tmdb-"+id+"}"}/{n} ({y}) {"{tmdb-"+id+"}"}`
	}
	if strings.TrimSpace(seriesFmt) == "" {
		seriesFmt = `Series/{az}/{n} ({y}) {"{tvdb-"+id+"}"}/{episode.special ? "Especiales" : "Temporada "+s00}/{n} ({y}) - {s00e00} - {t}`
	}

	tmdb := tmdbToken(meta)
	tvdb := tvdbToken(meta)
	yearStr := maybeInt(meta.Year)
	nyStr := title
	if yearStr != "" {
		nyStr = fmt.Sprintf("%s (%s)", title, yearStr)
	}

	seasonNum := defaultInt(meta.Season, 1)
	episodeNum := defaultInt(meta.Episode, 1)
	seasonStr := fmt.Sprintf("%02d", seasonNum)
	s00e00 := fmt.Sprintf("S%02dE%02d", seasonNum, episodeNum)

	normalizeFmt := func(raw string, isSeries bool) string {
		out := raw
		out = strings.ReplaceAll(out, `{"{tmdb-"+id+"}"}`, "{tmdb}")
		out = strings.ReplaceAll(out, `{"{tmdb-" + id + "}"}`, "{tmdb}")
		out = strings.ReplaceAll(out, `{"{tvdb-"+id+"}"}`, "{tvdb}")
		out = strings.ReplaceAll(out, `{"{tvdb-" + id + "}"}`, "{tvdb}")
		out = strings.ReplaceAll(out, `{"{"+"tmdb-"+id+"}"}`, "{tmdb}")
		out = strings.ReplaceAll(out, `{"{"+"tvdb-"+id+"}"}`, "{tvdb}")
		out = strings.ReplaceAll(out, `{episode.special ? "Especiales" : "Temporada "+s00}`, "Temporada {season}")
		out = strings.ReplaceAll(out, `{episode.special ? "Especiales" : "Temporada " + s00}`, "Temporada {season}")
		out = strings.ReplaceAll(out, `{episode.special ? 'Especiales' : 'Temporada '+s00}`, "Temporada {season}")
		out = strings.ReplaceAll(out, `{s.pad(2)}`, "{season}")

		if isSeries {
			if strings.Contains(out, "id+") || strings.Contains(out, "episode.special") || strings.Contains(out, "{\"") {
				out = "Series/{alpha}/{series} ({year}) {tvdb}/Temporada {season}/{series} ({year}) - {episode}{episode_title_suffix}"
			}
		} else {
			if strings.Contains(out, "id+") || strings.Contains(out, "{\"") {
				out = "Peliculas/{vf}/{alpha}/{title} ({year}) {tmdb}/{title} ({year}) {tmdb}"
			}
		}
		return out
	}

	mapping := map[string]string{
		"title":                title,
		"series":               title,
		"n":                    title,
		"year":                 yearStr,
		"y":                    yearStr,
		"ny":                   nyStr,
		"season":               seasonStr,
		"s":                    strconv.Itoa(seasonNum),
		"s00":                  seasonStr,
		"s.pad(2)":             seasonStr,
		"episode":              episodeToken(meta.Season, meta.Episode),
		"s00e00":               s00e00,
		"episode_title":        strings.TrimSpace(meta.ResolvedEpisodeTitle),
		"episode_title_suffix": episodeTitleSuffix(meta.ResolvedEpisodeTitle),
		"t":                    strings.TrimSpace(meta.ResolvedEpisodeTitle),
		"quality":              quality,
		"vf":                   quality,
		"alpha":                alpha,
		"az":                   alpha,
		"n[0]":                 alpha,
		"plex":                 title,
		"tvdb":                 tvdb,
		"tmdb":                 tmdb,
		"id":                   idToken(meta),
		"tmdbid":               tmdb,
		"tvdbid":               tvdb,
	}

	format := normalizeFmt(movieFmt, false)
	if kind == "series" {
		format = normalizeFmt(seriesFmt, true)
	}
	resolved := applyTokenFormat(format, mapping)
	return &FileBotResolveResult{
		RelativePath: filepath.ToSlash(strings.Trim(resolved, "/ ")),
		Method:       "fallback",
		EpisodeTitle: strings.TrimSpace(meta.ResolvedEpisodeTitle),
		TVDBID:       meta.TVDBID,
		TMDBID:       meta.TMDBID,
	}
}

func applyTokenFormat(format string, mapping map[string]string) string {
	out := format
	for k, v := range mapping {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	out = strings.ReplaceAll(out, "()", "")
	out = strings.ReplaceAll(out, "[]", "")
	out = strings.ReplaceAll(out, "{}", "")
	parts := strings.Split(out, "/")
	cleanParts := make([]string, 0, len(parts))
	for _, p := range parts {
		cleaned := strings.Join(strings.Fields(p), " ")
		if cleaned != "" {
			cleanParts = append(cleanParts, cleaned)
		}
	}
	return strings.Join(cleanParts, "/")
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
		if len(parts) > 1 {
			p1 := strings.ToLower(parts[1])
			qLow := strings.ToLower(q)
			if p1 == qLow || strings.TrimSuffix(p1, "p") == strings.TrimSuffix(qLow, "p") {
				return rel
			}
			if p1 == "unknown" || p1 == "{vf}" || p1 == "c" {
				parts[1] = q
				return filepath.ToSlash(strings.Join(parts, "/"))
			}
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

func detectTMDBIDFromPath(rel string) int {
	re := regexp.MustCompile(`\{tmdb-(\d+)\}`)
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
	if strings.Contains(last, ".") && len(filepath.Ext(last)) >= 2 && len(filepath.Ext(last)) <= 5 {
		return strings.Join(parts[:len(parts)-1], "/")
	}
	if len(parts) >= 2 && parts[len(parts)-1] == parts[len(parts)-2] {
		return strings.Join(parts[:len(parts)-1], "/")
	}
	if len(parts) >= 3 && (regexp.MustCompile(`(?i)(?:^|[\s\.\-_])S\d+E\d+(?:[\s\.\-_]|$)`).MatchString(last) || regexp.MustCompile(`(?i)(?:^|[\s\.\-_])\d+x\d+(?:[\s\.\-_]|$)`).MatchString(last)) {
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
	return ""
}

func tmdbToken(meta ItemMetadata) string {
	if meta.TMDBID > 0 {
		return fmt.Sprintf("{tmdb-%d}", meta.TMDBID)
	}
	return ""
}

func idToken(meta ItemMetadata) string {
	if meta.TMDBID > 0 {
		return strconv.Itoa(meta.TMDBID)
	}
	if meta.TVDBID > 0 {
		return strconv.Itoa(meta.TVDBID)
	}
	return ""
}

func episodeTitleSuffix(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	return " - " + title
}
