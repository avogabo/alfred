package geoffrey

import (
	"fmt"
	"strings"

	"alfred/internal/plex"
)

func (a *App) ResolveLibrarySection(input string) (plex.Library, error) {
	libs, err := a.plex.Libraries()
	if err != nil {
		return plex.Library{}, err
	}
	needle := strings.TrimSpace(input)
	if needle == "" {
		needle = strings.TrimSpace(a.memory.Data.UserPreferences.DefaultMovieLibrary)
	}
	for _, lib := range libs {
		if lib.Key == needle || strings.EqualFold(strings.TrimSpace(lib.Title), needle) {
			return lib, nil
		}
	}
	return plex.Library{}, fmt.Errorf("library %q not found", needle)
}
