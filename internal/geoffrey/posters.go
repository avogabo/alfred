package geoffrey

import (
	"fmt"
	"net/url"
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

		// TMDb Multi search (movies & series in Spanish)
		if multi, err := a.tmdb.SearchMulti(searchQuery); err == nil {
			for _, item := range multi {
				if item.PosterPath != "" {
					fullURL := "https://image.tmdb.org/t/p/w500" + item.PosterPath
					if !seen[fullURL] {
						seen[fullURL] = true
						nameText := item.Title
						if nameText == "" {
							nameText = item.Name
						}
						suggestions = append(suggestions, PosterSuggestion{
							URL:    fullURL,
							Source: "tmdb_title",
							Label:  fmt.Sprintf("Póster oficial TMDb: %s", nameText),
						})
					}
				}
				if len(suggestions) >= 6 {
					break
				}
			}
		}
	}

	// 2. Pollinations.ai (Free automatic AI-generated poster based on title & keywords)
	aiSubject := searchQuery
	if cleanTitle != "" && cleanPrompt != "" && !strings.EqualFold(cleanTitle, cleanPrompt) {
		aiSubject = cleanTitle + " " + cleanPrompt
	}
	if aiSubject != "" {
		cleanAI := strings.NewReplacer(",", " ", "(", " ", ")", " ", "\"", " ", "/", " ").Replace(aiSubject)
		cleanAI = strings.Join(strings.Fields(cleanAI), " ")

		promptCinematic := fmt.Sprintf("cinematic movie poster %s masterpiece", cleanAI)
		urlCinematic := fmt.Sprintf("https://image.pollinations.ai/prompt/%s?model=flux&width=600&height=900&nologo=true", url.PathEscape(promptCinematic))
		suggestions = append(suggestions, PosterSuggestion{
			URL:    urlCinematic,
			Source: "ai_pollinations_cinematic",
			Label:  "Póster cinematográfico IA (Pollinations Flux)",
		})

		promptArt := fmt.Sprintf("artistic minimalist movie poster %s illustration", cleanAI)
		urlArt := fmt.Sprintf("https://image.pollinations.ai/prompt/%s?model=flux&width=600&height=900&nologo=true", url.PathEscape(promptArt))
		suggestions = append(suggestions, PosterSuggestion{
			URL:    urlArt,
			Source: "ai_pollinations_artistic",
			Label:  "Póster artístico minimalista IA (Pollinations Flux)",
		})
	}

	return suggestions, nil
}

