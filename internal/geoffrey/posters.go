package geoffrey

import (
	"fmt"
	"strings"
)

func (a *App) ApplyCollectionPoster(sectionKey, collectionName, posterURL, posterBase64 string) error {
	if strings.TrimSpace(posterURL) == "" && strings.TrimSpace(posterBase64) == "" {
		return nil
	}
	if sectionKey == "" || sectionKey == "all" {
		libs, err := a.plex.Libraries()
		if err != nil {
			return err
		}
		for _, lib := range libs {
			_ = a.ApplyCollectionPoster(lib.Key, collectionName, posterURL, posterBase64)
		}
		return nil
	}
	collections, err := a.plex.ListCollections(sectionKey)
	if err != nil {
		return err
	}
	for _, item := range collections {
		if strings.EqualFold(strings.TrimSpace(item.Title), strings.TrimSpace(collectionName)) {
			if strings.TrimSpace(posterBase64) != "" {
				return a.plex.UploadPosterData(item.RatingKey, posterBase64)
			}
			return a.plex.SetPosterURL(item.RatingKey, posterURL)
		}
	}
	return fmt.Errorf("collection %q not found for poster apply", collectionName)
}

type PosterSuggestion struct {
	URL    string `json:"url"`
	Source string `json:"source"`
	Label  string `json:"label"`
}

func (a *App) SuggestPosters(title, prompt string) ([]PosterSuggestion, error) {
	var suggestions []PosterSuggestion
	seen := map[string]bool{}

	cleanTitle := strings.TrimSpace(title)
	cleanPrompt := strings.TrimSpace(prompt)
	searchQuery := cleanTitle
	if searchQuery == "" {
		searchQuery = cleanPrompt
	}

	// 1. TMDb Collections (Official Saga/Franchise Posters)
	if a.tmdb != nil && a.tmdb.Enabled() && searchQuery != "" {
		if collections, err := a.tmdb.SearchCollection(searchQuery); err == nil {
			for _, col := range collections {
				if col.PosterPath != "" {
					fullURL := "https://image.tmdb.org/t/p/w500" + col.PosterPath
					if !seen[fullURL] {
						seen[fullURL] = true
						suggestions = append(suggestions, PosterSuggestion{
							URL:    fullURL,
							Source: "tmdb_collection",
							Label:  fmt.Sprintf("Saga oficial TMDb: %s", col.Name),
						})
					}
				}
				if len(suggestions) >= 3 {
					break
				}
			}
		}

		// TMDb Movie Poster fallback if few collection posters
		if len(suggestions) < 3 {
			if movies, err := a.tmdb.SearchMovie(searchQuery); err == nil {
				for _, m := range movies {
					if m.PosterPath != "" {
						fullURL := "https://image.tmdb.org/t/p/w500" + m.PosterPath
						if !seen[fullURL] {
							seen[fullURL] = true
							suggestions = append(suggestions, PosterSuggestion{
								URL:    fullURL,
								Source: "tmdb_movie",
								Label:  fmt.Sprintf("Película TMDb: %s", m.Title),
							})
						}
					}
					if len(suggestions) >= 4 {
						break
					}
				}
			}
		}
	}

	// 2. Pollinations.ai (Free automatic AI-generated poster based on title & keywords)
	aiSubject := searchQuery
	if cleanTitle != "" && cleanPrompt != "" && !strings.EqualFold(cleanTitle, cleanPrompt) {
		aiSubject = cleanTitle + " (" + cleanPrompt + ")"
	}
	if aiSubject != "" {
		promptCinematic := fmt.Sprintf("cinematic movie collection poster, %s, movie title typography, dramatic film lighting, official studio theatrical poster, 8k", aiSubject)
		urlCinematic := fmt.Sprintf("https://image.pollinations.ai/prompt/%s?width=600&height=900&nologo=true", strings.ReplaceAll(promptCinematic, " ", "%20"))
		suggestions = append(suggestions, PosterSuggestion{
			URL:    urlCinematic,
			Source: "ai_pollinations_cinematic",
			Label:  "Póster cinematográfico IA (Pollinations)",
		})

		promptArt := fmt.Sprintf("minimalist vintage movie poster art for %s, retro graphic screenprint, vibrant cinema illustration, clean design", aiSubject)
		urlArt := fmt.Sprintf("https://image.pollinations.ai/prompt/%s?width=600&height=900&nologo=true", strings.ReplaceAll(promptArt, " ", "%20"))
		suggestions = append(suggestions, PosterSuggestion{
			URL:    urlArt,
			Source: "ai_pollinations_artistic",
			Label:  "Póster artístico minimalista IA (Pollinations)",
		})
	}

	return suggestions, nil
}

