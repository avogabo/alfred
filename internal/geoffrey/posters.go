package geoffrey

import (
	"fmt"
	"strings"
)

func (a *App) ApplyCollectionPoster(sectionKey, collectionName, posterURL, posterBase64 string) error {
	if strings.TrimSpace(posterURL) == "" && strings.TrimSpace(posterBase64) == "" {
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
