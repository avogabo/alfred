package winston

import (
	"context"
	"path/filepath"
	"strings"
)

func (p *ImportProcessor) BuildPreview(sourceNZB string, meta ItemMetadata) *ItemPreview {
	preview := &ItemPreview{
		SourceNZBPath: sourceNZB,
		Metadata:      meta,
		Kind:          normalizeKind(meta.Kind),
		Confidence:    ConfidenceLow,
		State:         StateDetected,
	}

	base := strings.TrimSuffix(filepath.Base(sourceNZB), filepath.Ext(sourceNZB))
	base = stripSyntheticTestPrefixes(base)
	if preview.Kind == "" || preview.Kind == "auto" {
		preview.Kind = guessKindFromPath(sourceNZB)
	}
	if preview.Metadata.Title == "" {
		preview.Metadata.Title = cleanupTitle(base)
	}
	if strings.TrimSpace(preview.Metadata.Quality) == "" {
		preview.Metadata.Quality = detectQualityForSource(sourceNZB)
	}

	resolved, confidence, candidates, reason := p.matcher.Resolve(preview.Metadata, sourceNZB)
	preview.Metadata = resolved
	preview.Kind = normalizeKind(preview.Metadata.Kind)
	preview.Confidence = confidence
	preview.Reason = reason
	preview.Candidates = candidates

	preview.ProposedPath = p.buildPathForPreview(sourceNZB, preview)
	if preview.ResolverMethod == "filebot" {
		preview.Confidence = ConfidenceHigh
		if preview.Reason == "" || preview.Reason == "name_parse" {
			preview.Reason = "filebot_match"
		}
	}
	if preview.Confidence == ConfidenceLow {
		preview.State = StateNeedsReview
	} else {
		preview.State = StateApproved
	}
	return preview
}

func (p *ImportProcessor) buildPathForPreview(sourceNZB string, preview *ItemPreview) string {
	if preview.Metadata.RelativePathOverride != "" {
		return preview.Metadata.RelativePathOverride
	}
	if strings.TrimSpace(preview.Metadata.ResolvedRelativePath) != "" {
		preview.ResolverMethod = "cached"
		return preview.Metadata.ResolvedRelativePath
	}
	if p.filebot != nil && p.filebot.Enabled() {
		if resolved, err := p.filebot.Resolve(context.Background(), sourceNZB, preview.Metadata); err == nil && resolved != nil && strings.TrimSpace(resolved.RelativePath) != "" {
			preview.ResolverMethod = resolved.Method
			preview.Metadata.ResolvedRelativePath = strings.TrimSpace(resolved.RelativePath)
			if preview.Metadata.ResolvedEpisodeTitle == "" && strings.TrimSpace(resolved.EpisodeTitle) != "" {
				preview.Metadata.ResolvedEpisodeTitle = strings.TrimSpace(resolved.EpisodeTitle)
			}
			if preview.Metadata.TVDBID == 0 && resolved.TVDBID > 0 {
				preview.Metadata.TVDBID = resolved.TVDBID
			}
			if preview.Metadata.TMDBID == 0 && resolved.TMDBID > 0 {
				preview.Metadata.TMDBID = resolved.TMDBID
			}
			return resolved.RelativePath
		} else if err != nil {
			preview.ResolverMethod = "fallback"
			preview.ResolverError = err.Error()
		}
	}
	if preview.ResolverMethod == "" {
		preview.ResolverMethod = "fallback"
	}
	return p.buildRelativePath(sourceNZB)
}

func normalizeKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	switch k {
	case "movie", "series", "episode", "auto":
		return k
	default:
		return "auto"
	}
}

func guessKindFromPath(path string) string {
	low := strings.ToLower(path)
	if strings.Contains(low, "season") || strings.Contains(low, "temporada") || strings.Contains(low, "series") {
		return "series"
	}
	return "movie"
}

func cleanupTitle(s string) string {
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

func stripSyntheticTestPrefixes(s string) string {
	s = strings.TrimSpace(s)
	prefixes := []string{
		"TEST-WINSTON-REAL-",
		"TEST-WINSTON-NORMAL-",
		"TEST-WINSTON-NUEVO-",
		"TEST-WINSTON-",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(s, prefix))
		}
	}
	return s
}
