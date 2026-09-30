package geoffrey

import (
	"fmt"
	"strings"
	"time"

	"alfred/internal/geoffrey/memory"
	"alfred/internal/plex"
)

func (a *App) CreateCollectionFromTitles(sectionKey, collectionName string, titles []string, sourcePrompt string, temporary bool, expiresAt string) error {
	if strings.TrimSpace(collectionName) == "" {
		return fmt.Errorf("collection name is required")
	}
	if len(titles) == 0 {
		return fmt.Errorf("at least one title is required")
	}
	var ratingKeys []string
	for _, title := range titles {
		results, err := a.plex.Search(sectionKey, title)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			return fmt.Errorf("no plex results found for %q", title)
		}
		ratingKeys = append(ratingKeys, results[0].RatingKey)
	}
	if err := a.plex.CreateCollection(sectionKey, collectionName, ratingKeys); err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	a.memory.Data.History = append(a.memory.Data.History, memory.CollectionRecord{
		Name:         collectionName,
		Library:      sectionKey,
		SourcePrompt: sourcePrompt,
		Temporary:    temporary,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	return a.memory.Save()
}

func (a *App) DeleteCollectionByName(sectionKey, collectionName string) error {
	if sectionKey == "" || sectionKey == "all" {
		libs, err := a.plex.Libraries()
		if err != nil {
			return err
		}
		deletedAny := false
		var lastErr error
		for _, lib := range libs {
			if collections, err := a.plex.ListCollections(lib.Key); err == nil {
				for _, item := range collections {
					if strings.EqualFold(strings.TrimSpace(item.Title), strings.TrimSpace(collectionName)) {
						if err := a.plex.DeleteCollection(item.RatingKey); err == nil {
							deletedAny = true
						} else {
							lastErr = err
						}
					}
				}
			}
		}
		if deletedAny {
			return nil
		}
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("collection %q not found", collectionName)
	}
	collections, err := a.plex.ListCollections(sectionKey)
	if err != nil {
		return err
	}
	for _, item := range collections {
		if strings.EqualFold(strings.TrimSpace(item.Title), strings.TrimSpace(collectionName)) {
			return a.plex.DeleteCollection(item.RatingKey)
		}
	}
	return fmt.Errorf("collection %q not found", collectionName)
}

func (a *App) Libraries() ([]plex.Library, error) {
	return a.plex.Libraries()
}
