package geoffrey

import (
	"fmt"
	"strings"
)

func (a *App) handleTelegramAction(text string) string {
	trimmed := strings.TrimSpace(text)
	low := strings.ToLower(trimmed)

	if strings.HasPrefix(low, "/search ") {
		payload := strings.TrimSpace(trimmed[len("/search "):])
		parts := strings.SplitN(payload, "|", 2)
		if len(parts) != 2 {
			return "Uso: /search <biblioteca>|<texto>"
		}
		lib, err := a.ResolveLibrarySection(strings.TrimSpace(parts[0]))
		if err != nil {
			return "No encuentro la biblioteca: " + err.Error()
		}
		items, err := a.Search(lib.Key, strings.TrimSpace(parts[1]))
		if err != nil {
			return "Error buscando: " + err.Error()
		}
		if len(items) == 0 {
			return "No encontré resultados."
		}
		var lines []string
		for i, item := range items {
			if i >= 10 {
				break
			}
			lines = append(lines, fmt.Sprintf("- %s (%d)", item.Title, item.Year))
		}
		return strings.Join(lines, "\n")
	}

	if strings.HasPrefix(low, "/create_collection ") {
		payload := strings.TrimSpace(trimmed[len("/create_collection "):])
		parts := strings.SplitN(payload, "|", 3)
		if len(parts) != 3 {
			return "Uso: /create_collection <biblioteca>|<nombre>|<titulo1, titulo2>"
		}
		lib, err := a.ResolveLibrarySection(strings.TrimSpace(parts[0]))
		if err != nil {
			return "No encuentro la biblioteca: " + err.Error()
		}
		name := strings.TrimSpace(parts[1])
		titles := SplitCSV(parts[2])
		if err := a.CreateCollectionFromTitles(lib.Key, name, titles, trimmed, false, ""); err != nil {
			return "No pude crear la colección: " + err.Error()
		}
		return fmt.Sprintf("Colección creada: %s", name)
	}

	if strings.HasPrefix(low, "/create_temporary_collection ") {
		payload := strings.TrimSpace(trimmed[len("/create_temporary_collection "):])
		parts := strings.SplitN(payload, "|", 4)
		if len(parts) < 3 {
			return "Uso: /create_temporary_collection <biblioteca>|<nombre>|<titulo1, titulo2>|[expiresAt RFC3339]"
		}
		lib, err := a.ResolveLibrarySection(strings.TrimSpace(parts[0]))
		if err != nil {
			return "No encuentro la biblioteca: " + err.Error()
		}
		name := strings.TrimSpace(parts[1])
		titles := SplitCSV(parts[2])
		expiresAt := ""
		if len(parts) == 4 {
			expiresAt = strings.TrimSpace(parts[3])
		}
		if err := a.CreateTemporaryCollectionFromTitles(lib.Key, name, titles, trimmed, expiresAt); err != nil {
			return "No pude crear la colección temporal: " + err.Error()
		}
		return fmt.Sprintf("Colección temporal creada: %s", name)
	}

	if strings.HasPrefix(low, "/delete_collection ") {
		payload := strings.TrimSpace(trimmed[len("/delete_collection "):])
		parts := strings.SplitN(payload, "|", 2)
		if len(parts) != 2 {
			return "Uso: /delete_collection <biblioteca>|<nombre>"
		}
		lib, err := a.ResolveLibrarySection(strings.TrimSpace(parts[0]))
		if err != nil {
			return "No encuentro la biblioteca: " + err.Error()
		}
		name := strings.TrimSpace(parts[1])
		if err := a.RequestDelete(lib.Key, name); err != nil {
			return "No pude preparar el borrado: " + err.Error()
		}
		return fmt.Sprintf("Confirma con /confirm_delete %s|%s", strings.TrimSpace(parts[0]), name)
	}

	if strings.HasPrefix(low, "/confirm_delete ") {
		payload := strings.TrimSpace(trimmed[len("/confirm_delete "):])
		parts := strings.SplitN(payload, "|", 2)
		if len(parts) != 2 {
			return "Uso: /confirm_delete <biblioteca>|<nombre>"
		}
		lib, err := a.ResolveLibrarySection(strings.TrimSpace(parts[0]))
		if err != nil {
			return "No encuentro la biblioteca: " + err.Error()
		}
		name := strings.TrimSpace(parts[1])
		if err := a.ConfirmDelete(lib.Key, name); err != nil {
			return "No pude borrar la colección: " + err.Error()
		}
		return fmt.Sprintf("Colección borrada: %s", name)
	}

	return ""
}
