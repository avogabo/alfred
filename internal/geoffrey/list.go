package geoffrey

import (
	"alfred/internal/plex"
)

func (a *App) Search(sectionKey, query string) ([]plex.Video, error) {
	return a.plex.Search(sectionKey, query)
}

func (a *App) Collections(sectionKey string) ([]plex.Collection, error) {
	return a.plex.ListCollections(sectionKey)
}
