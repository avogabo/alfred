package winston

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Matcher struct{}

func NewMatcher() *Matcher { return &Matcher{} }

var (
	reEpisodeA = regexp.MustCompile(`(?i)^(?P<title>.+?)\s*\((?P<year>\d{4})\)\s*(?P<season>\d{1,2})x(?P<episode>\d{1,2})$`)
	reEpisodeB = regexp.MustCompile(`(?i)^(?P<title>.+?)\s*[. _-]*S(?P<season>\d{1,2})E(?P<episode>\d{1,2})(?:\s*\((?P<year>\d{4})\))?$`)
	reSeasonA  = regexp.MustCompile(`(?i)^(?P<title>.+?)\s*\((?P<year>\d{4})\)\s*Temporada\s*(?P<season>\d{1,2})$`)
	reSeasonB  = regexp.MustCompile(`(?i)^(?P<title>.+?)\s*\((?P<year>\d{4})\)\s*Season\s*(?P<season>\d{1,2})$`)
	reMovie    = regexp.MustCompile(`(?i)^(?P<title>.+?)\s*\((?P<year>\d{4})\)$`)
)

func (m *Matcher) Resolve(meta ItemMetadata, sourceNZB string) (ItemMetadata, MatchConfidence, []CandidateMatch, string) {
	if meta.RelativePathOverride != "" {
		return meta, ConfidenceHigh, nil, "relative_path_override"
	}
	if meta.TMDBID > 0 || meta.TVDBID > 0 || meta.IMDBID != "" {
		return meta, ConfidenceHigh, nil, "explicit_id"
	}
	if meta.Title != "" && meta.Year > 0 {
		return meta, ConfidenceMedium, nil, "manual_title_year"
	}

	base := cleanupTitle(stripSyntheticTestPrefixes(strings.TrimSuffix(filepathBase(sourceNZB), filepathExt(sourceNZB))))
	if parsed, ok := parseStructuredName(base, meta); ok {
		reason := "title_year_episode_parse"
		score := 78
		if parsed.Kind == "series" && parsed.Season > 0 && parsed.Episode == 0 {
			reason = "title_year_season_parse"
			score = 74
		}
		candidates := []CandidateMatch{{
			Label:  parsed.Title,
			Kind:   normalizeKind(parsed.Kind),
			Year:   parsed.Year,
			Reason: reason,
			Score:  score,
		}}
		return parsed, ConfidenceMedium, candidates, reason
	}

	meta.Title = base
	return meta, ConfidenceLow, []CandidateMatch{{Label: base, Kind: normalizeKind(meta.Kind), Reason: "name_parse", Score: 40}}, "name_parse"
}

func parseStructuredName(base string, meta ItemMetadata) (ItemMetadata, bool) {
	for _, re := range []*regexp.Regexp{reEpisodeA, reEpisodeB} {
		if out, ok := parseEpisodePattern(re, base, meta); ok {
			return out, true
		}
	}
	if out, ok := parseMoviePattern(base, meta); ok {
		return out, true
	}
	if out, ok := parseSeasonPattern(base, meta); ok {
		return out, true
	}
	return meta, false
}

func parseSeasonPattern(base string, meta ItemMetadata) (ItemMetadata, bool) {
	for _, re := range []*regexp.Regexp{reSeasonA, reSeasonB} {
		match := re.FindStringSubmatch(base)
		if match == nil {
			continue
		}
		groups := map[string]string{}
		for i, name := range re.SubexpNames() {
			if i > 0 && name != "" {
				groups[name] = strings.TrimSpace(match[i])
			}
		}
		meta.Title = cleanupTitle(groups["title"])
		meta.Kind = "series"
		meta.Year = parseIntOr(groups["year"], meta.Year)
		meta.Season = parseIntOr(groups["season"], meta.Season)
		return meta, meta.Title != "" && meta.Year > 0 && meta.Season > 0
	}
	return meta, false
}

func parseEpisodePattern(re *regexp.Regexp, base string, meta ItemMetadata) (ItemMetadata, bool) {
	match := re.FindStringSubmatch(base)
	if match == nil {
		return meta, false
	}
	groups := map[string]string{}
	for i, name := range re.SubexpNames() {
		if i > 0 && name != "" {
			groups[name] = strings.TrimSpace(match[i])
		}
	}
	meta.Title = cleanupTitle(groups["title"])
	meta.Kind = "series"
	meta.Year = parseIntOr(groups["year"], meta.Year)
	meta.Season = parseIntOr(groups["season"], meta.Season)
	meta.Episode = parseIntOr(groups["episode"], meta.Episode)
	return meta, meta.Title != "" && meta.Season > 0 && meta.Episode > 0
}

func parseMoviePattern(base string, meta ItemMetadata) (ItemMetadata, bool) {
	match := reMovie.FindStringSubmatch(base)
	if match == nil {
		return meta, false
	}
	groups := map[string]string{}
	for i, name := range reMovie.SubexpNames() {
		if i > 0 && name != "" {
			groups[name] = strings.TrimSpace(match[i])
		}
	}
	meta.Title = cleanupTitle(groups["title"])
	meta.Kind = "movie"
	meta.Year = parseIntOr(groups["year"], meta.Year)
	return meta, meta.Title != "" && meta.Year > 0
}

func parseIntOr(s string, fallback int) int {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fallback
	}
	return v
}

func filepathBase(path string) string {
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}

func filepathExt(path string) string {
	base := filepathBase(path)
	idx := strings.LastIndex(base, ".")
	if idx < 0 {
		return ""
	}
	return base[idx:]
}

func debugCandidate(c CandidateMatch) string {
	return fmt.Sprintf("%s/%s/%d", c.Label, c.Kind, c.Year)
}
