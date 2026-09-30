package geoffrey

import (
	"alfred/internal/plex"
)

func (a *App) Search(sectionKey, query string) ([]plex.Video, error) {
	if sectionKey == "" || sectionKey == "all" {
		libs, err := a.plex.Libraries()
		if err != nil {
			return nil, err
		}
		var allResults []plex.Video
		seen := map[string]bool{}
		for _, lib := range libs {
			if items, err := a.plex.Search(lib.Key, query); err == nil {
				for _, it := range items {
					if !seen[it.RatingKey] {
						seen[it.RatingKey] = true
						it.LibraryKey = lib.Key
						it.LibraryTitle = lib.Title
						allResults = append(allResults, it)
					}
				}
			}
		}
		return allResults, nil
	}
	items, err := a.plex.Search(sectionKey, query)
	if err == nil {
		libTitle := ""
		if libs, lerr := a.plex.Libraries(); lerr == nil {
			for _, l := range libs {
				if l.Key == sectionKey {
					libTitle = l.Title
					break
				}
			}
		}
		for i := range items {
			items[i].LibraryKey = sectionKey
			items[i].LibraryTitle = libTitle
		}
	}
	return items, err
}

func (a *App) Collections(sectionKey string) ([]plex.Collection, error) {
	if sectionKey == "" || sectionKey == "all" {
		libs, err := a.plex.Libraries()
		if err != nil {
			return nil, err
		}
		var all []plex.Collection
		seen := map[string]bool{}
		for _, lib := range libs {
			if cols, err := a.plex.ListCollections(lib.Key); err == nil {
				for _, col := range cols {
					k := col.Title
					if !seen[k] {
						seen[k] = true
						all = append(all, col)
					}
				}
			}
		}
		return all, nil
	}
	return a.plex.ListCollections(sectionKey)
}
