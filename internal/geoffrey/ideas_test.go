package geoffrey

import (
	"testing"

	"alfred/internal/config"
	"alfred/internal/plex"
	"alfred/internal/tmdb"
)

func TestExtractFacets(t *testing.T) {
	app := &App{cfg: config.Config{}}

	tests := []struct {
		input            string
		expectedGenre    int
		expectedKW       int
		expectedYearGte  int
		expectedYearLte  int
		expectedExactYr  int
	}{
		{
			input:           "comedias navideñas",
			expectedGenre:   35,
			expectedKW:      207317,
			expectedYearGte: 0,
			expectedYearLte: 0,
		},
		{
			input:           "películas de terror de los 80",
			expectedGenre:   27,
			expectedYearGte: 1980,
			expectedYearLte: 1989,
		},
		{
			input:           "ciencia ficcion de 1999",
			expectedGenre:   878,
			expectedExactYr: 1999,
		},
		{
			input:         "zombies y comedia",
			expectedGenre: 35,
			expectedKW:    12377,
		},
	}

	for _, tc := range tests {
		facets := app.extractFacets(tc.input)
		t.Logf("Input: '%s' -> Genres: %v, KWs: %v, Range: [%d..%d], Exact: %d",
			tc.input, facets.GenreIDs, facets.ThematicKWIDs, facets.YearGte, facets.YearLte, facets.ExactYear)

		if tc.expectedGenre > 0 {
			found := false
			for _, g := range facets.GenreIDs {
				if g == tc.expectedGenre {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("[%s] expected genre %d in %v", tc.input, tc.expectedGenre, facets.GenreIDs)
			}
		}

		if tc.expectedKW > 0 {
			found := false
			for _, kw := range facets.ThematicKWIDs {
				if kw == tc.expectedKW {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("[%s] expected keyword %d in %v", tc.input, tc.expectedKW, facets.ThematicKWIDs)
			}
		}

		if tc.expectedYearGte > 0 && facets.YearGte != tc.expectedYearGte {
			t.Errorf("[%s] expected YearGte %d, got %d", tc.input, tc.expectedYearGte, facets.YearGte)
		}
		if tc.expectedYearLte > 0 && facets.YearLte != tc.expectedYearLte {
			t.Errorf("[%s] expected YearLte %d, got %d", tc.input, tc.expectedYearLte, facets.YearLte)
		}
		if tc.expectedExactYr > 0 && facets.ExactYear != tc.expectedExactYr {
			t.Errorf("[%s] expected ExactYear %d, got %d", tc.input, tc.expectedExactYr, facets.ExactYear)
		}
	}
}

func TestScoreLibraryVideo(t *testing.T) {
	app := &App{cfg: config.Config{}}
	facets := app.extractFacets("comedias navideñas")

	tmdbByID := map[int]tmdb.Movie{
		771: {ID: 771, Title: "Solo en casa", OriginalTitle: "Home Alone", Release: "1990-11-16"},
		10719: {ID: 10719, Title: "Elf", OriginalTitle: "Elf", Release: "2003-11-07"},
	}
	tmdbByTitle := map[string]int{
		"solo en casa:1990": 771,
		"home alone:1990":   771,
		"elf:2003":          10719,
	}

	// 1. Exact TMDb ID match
	vid1 := plex.Video{
		RatingKey: "101",
		Title:     "Solo en casa",
		Year:      1990,
		TMDBID:    771,
	}
	score1, reason1 := scoreLibraryVideo(vid1, facets, tmdbByID, tmdbByTitle)
	if score1 != 100 {
		t.Errorf("Expected score 100 for exact TMDB ID match, got %d (reason: %s)", score1, reason1)
	}

	// 2. Title + Year match without TMDBID
	vid2 := plex.Video{
		RatingKey: "102",
		Title:     "Elf",
		Year:      2003,
		TMDBID:    0,
	}
	score2, reason2 := scoreLibraryVideo(vid2, facets, tmdbByID, tmdbByTitle)
	if score2 != 95 {
		t.Errorf("Expected score 95 for title match, got %d (reason: %s)", score2, reason2)
	}

	// 3. Local Library Match: Genre Comedia + Summary contains "Navidad" (not in TMDb list)
	vid3 := plex.Video{
		RatingKey: "103",
		Title:     "Una comedia española desconocida",
		Year:      2021,
		Genres:    []string{"Comedia"},
		Summary:   "Tres hermanos regresan a su pueblo en Navidad para repartir la herencia de su abuela.",
	}
	score3, reason3 := scoreLibraryVideo(vid3, facets, tmdbByID, tmdbByTitle)
	if score3 != 85 {
		t.Errorf("Expected score 85 for genre+summary match, got %d (reason: %s)", score3, reason3)
	}

	// 4. Non-matching movie (Drama with no Christmas theme)
	vid4 := plex.Video{
		RatingKey: "104",
		Title:     "Oppenheimer",
		Year:      2023,
		Genres:    []string{"Drama", "Historia"},
		Summary:   "La historia del científico estadounidense J. Robert Oppenheimer y su papel en el Proyecto Manhattan.",
	}
	score4, _ := scoreLibraryVideo(vid4, facets, tmdbByID, tmdbByTitle)
	if score4 != 0 {
		t.Errorf("Expected score 0 for Oppenheimer in comedias navideñas, got %d", score4)
	}
}
