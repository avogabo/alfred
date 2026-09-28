package geoffrey

import (
	"fmt"
	"strings"
)

func (a *App) handleNaturalCollectionIntent(text string) string {
	low := strings.ToLower(strings.TrimSpace(text))

	if strings.Contains(low, "coleccion") || strings.Contains(low, "colección") {
		if strings.Contains(low, "gorila") || strings.Contains(low, "gorilas") {
			return "Puedo hacerla. Prueba ya con: /create_collection Películas|Gorilas|King Kong, Gorilas en la niebla"
		}
		if strings.Contains(low, "halloween") {
			return "Puedo hacer una temporal. Prueba: /create_temporary_collection Películas|Halloween de risa|Scary Movie, Gremlins"
		}
		if strings.Contains(low, "navidad") {
			return "Puedo montarte una base de Navidad TV. Si quieres, el siguiente paso es enseñarle a Geoffrey a resolver recetas completas automáticamente."
		}
		return fmt.Sprintf("Entiendo que quieres trabajar con colecciones, pero aún necesito un comando algo guiado. Usa /recipes o /create_collection.")
	}

	return ""
}
