package geoffrey

import (
	"fmt"
	"time"

	"alfred/internal/geoffrey/memory"
)

func (a *App) CreateTemporaryCollectionFromTitles(sectionKey, collectionName string, titles []string, sourcePrompt string, expiresAt string) error {
	if expiresAt == "" {
		expiresAt = time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339)
	}
	return a.CreateCollectionFromTitles(sectionKey, collectionName, titles, sourcePrompt, true, expiresAt)
}

func (a *App) PendingDelete(collectionName string) bool {
	for _, item := range a.memory.Data.PendingDeletes {
		if item.Name == collectionName {
			return true
		}
	}
	return false
}

func (a *App) RequestDelete(library, collectionName string) error {
	if a.PendingDelete(collectionName) {
		return nil
	}
	a.memory.Data.PendingDeletes = append(a.memory.Data.PendingDeletes, memory.PendingDelete{
		Library:     library,
		Name:        collectionName,
		RequestedAt: time.Now().Format(time.RFC3339),
	})
	return a.memory.Save()
}

func (a *App) ConfirmDelete(sectionKey, collectionName string) error {
	if err := a.DeleteCollectionByName(sectionKey, collectionName); err != nil {
		return err
	}
	filtered := a.memory.Data.PendingDeletes[:0]
	for _, item := range a.memory.Data.PendingDeletes {
		if item.Name == collectionName {
			continue
		}
		filtered = append(filtered, item)
	}
	a.memory.Data.PendingDeletes = filtered
	return a.memory.Save()
}

func (a *App) PendingDeletes() []string {
	out := make([]string, 0, len(a.memory.Data.PendingDeletes))
	for _, item := range a.memory.Data.PendingDeletes {
		out = append(out, fmt.Sprintf("%s|%s", item.Library, item.Name))
	}
	return out
}
