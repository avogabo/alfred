package winston

import (
	"testing"
)

func TestMatcher(t *testing.T) {
	matcher := NewMatcher()

	tests := []struct {
		nzb          string
		expectedKind string
		expectedName string
		minScore     int
		expectedConf MatchConfidence
	}{
		{
			nzb:          "The.Last.of.Us.S01E03.1080p.WEB.H264.nzb",
			expectedKind: "series",
			expectedName: "The Last of Us",
			minScore:     70,
			expectedConf: ConfidenceMedium,
		},
		{
			nzb:          "El.Internado.Temporada.1.1080p.nzb",
			expectedKind: "series",
			expectedName: "El Internado",
			minScore:     70,
			expectedConf: ConfidenceMedium,
		},
		{
			nzb:          "Oppenheimer.2023.2160p.UHD.BluRay.x265.nzb",
			expectedKind: "movie",
			expectedName: "Oppenheimer",
			minScore:     70,
			expectedConf: ConfidenceMedium,
		},
		{
			nzb:          "Blade.Runner.2049.2017.1080p.BluRay.x264.nzb",
			expectedKind: "movie",
			expectedName: "Blade Runner 2049",
			minScore:     70,
			expectedConf: ConfidenceMedium,
		},
	}

	for _, tc := range tests {
		meta, conf, candidates, reason := matcher.Resolve(ItemMetadata{}, tc.nzb)
		t.Logf("NZB: %s -> Title: '%s', Year: %d, Kind: %s, Conf: %s, Reason: %s",
			tc.nzb, meta.Title, meta.Year, meta.Kind, conf, reason)
		if conf != tc.expectedConf {
			t.Errorf("[%s] expected confidence %s, got %s", tc.nzb, tc.expectedConf, conf)
		}
		if meta.Kind != tc.expectedKind {
			t.Errorf("[%s] expected kind %s, got %s", tc.nzb, tc.expectedKind, meta.Kind)
		}
		if meta.Title != tc.expectedName {
			t.Errorf("[%s] expected title %s, got %s", tc.nzb, tc.expectedName, meta.Title)
		}
		if len(candidates) > 0 && candidates[0].Score < tc.minScore {
			t.Errorf("[%s] expected score >= %d, got %d", tc.nzb, tc.minScore, candidates[0].Score)
		}
	}
}
