package geoffrey

import (
	"log"
	"time"
)

func (a *App) StartSchedulers() {
	go a.expirationLoop()
}

func (a *App) expirationLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		a.cleanupExpiredCollections()
	}
}

func (a *App) cleanupExpiredCollections() {
	if len(a.memory.Data.History) == 0 {
		return
	}
	now := time.Now()
	changed := false
	kept := a.memory.Data.History[:0]
	for _, item := range a.memory.Data.History {
		if !item.Temporary || item.ExpiresAt == "" {
			kept = append(kept, item)
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339, item.ExpiresAt)
		if err != nil {
			kept = append(kept, item)
			continue
		}
		if now.Before(expiresAt) {
			kept = append(kept, item)
			continue
		}
		if err := a.DeleteCollectionByName(item.Library, item.Name); err != nil {
			log.Printf("geoffrey: failed to delete expired collection %s: %v", item.Name, err)
			kept = append(kept, item)
			continue
		}
		log.Printf("geoffrey: deleted expired temporary collection %s", item.Name)
		changed = true
	}
	if changed {
		a.memory.Data.History = kept
		if err := a.memory.Save(); err != nil {
			log.Printf("geoffrey: failed to save memory after expiration cleanup: %v", err)
		}
		return
	}
	a.memory.Data.History = kept
}
