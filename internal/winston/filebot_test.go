package winston

import (
	"context"
	"testing"

	"alfred/internal/config"
)

func TestFileBotRecipeFallback(t *testing.T) {
	cfg := config.Config{
		FileBotMovieFormat:  `Peliculas/{vf}/{az}/{n} ({y}) {"{tmdb-"+id+"}"}/{n} ({y}) {"{tmdb-"+id+"}"}`,
		FileBotSeriesFormat: `Series/{az}/{n} ({y}) {"{tvdb-"+id+"}"}/{episode.special ? "Especiales" : "Temporada "+s00}/{n} ({y}) - {s00e00} - {t}`,
		FileBotBinary:       "/nonexistent/filebot",
	}

	client := NewFileBotClient(cfg)

	t.Run("MovieWithTMDBIDAnd2160p", func(t *testing.T) {
		meta := ItemMetadata{
			Title:   "Oppenheimer",
			Year:    2023,
			Kind:    "movie",
			Quality: "2160",
			TMDBID:  872585,
		}
		res, err := client.Resolve(context.Background(), "Oppenheimer.2023.2160p.UHD.BluRay.x265.nzb", meta)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "Peliculas/2160/O/Oppenheimer (2023) {tmdb-872585}"
		if res.RelativePath != expected {
			t.Errorf("expected %q, got %q", expected, res.RelativePath)
		}
	})

	t.Run("MovieWithoutTMDBIDAnd1080p", func(t *testing.T) {
		meta := ItemMetadata{
			Title:   "Blade Runner 2049",
			Year:    2017,
			Kind:    "movie",
			Quality: "1080",
		}
		res, err := client.Resolve(context.Background(), "Blade.Runner.2049.2017.1080p.BluRay.x264.nzb", meta)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "Peliculas/1080/B/Blade Runner 2049 (2017)"
		if res.RelativePath != expected {
			t.Errorf("expected %q, got %q", expected, res.RelativePath)
		}
	})

	t.Run("SeriesWithTVDBID", func(t *testing.T) {
		meta := ItemMetadata{
			Title:                "The Last of Us",
			Year:                 2023,
			Kind:                 "series",
			Season:               1,
			Episode:              3,
			TVDBID:               100088,
			ResolvedEpisodeTitle: "Long, Long Time",
		}
		res, err := client.Resolve(context.Background(), "The.Last.of.Us.S01E03.1080p.WEB.H264.nzb", meta)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "Series/T/The Last of Us (2023) {tvdb-100088}/Temporada 01"
		if res.RelativePath != expected {
			t.Errorf("expected %q, got %q", expected, res.RelativePath)
		}
	})
}
