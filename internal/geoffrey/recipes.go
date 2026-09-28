package geoffrey

import (
	"fmt"
	"strings"

	"alfred/internal/geoffrey/memory"
)

func (a *App) SaveRecipe(recipe memory.Recipe) error {
	if strings.TrimSpace(recipe.ID) == "" {
		return fmt.Errorf("recipe id is required")
	}
	if strings.TrimSpace(recipe.Name) == "" {
		return fmt.Errorf("recipe name is required")
	}
	for i, existing := range a.memory.Data.Recipes {
		if existing.ID == recipe.ID {
			a.memory.Data.Recipes[i] = recipe
			return a.memory.Save()
		}
	}
	a.memory.Data.Recipes = append(a.memory.Data.Recipes, recipe)
	return a.memory.Save()
}

func (a *App) Recipes() []memory.Recipe {
	return append([]memory.Recipe{}, a.memory.Data.Recipes...)
}
