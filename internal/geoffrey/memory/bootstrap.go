package memory

func DefaultRecipes() []Recipe {
	return []Recipe{
		{
			ID:                "halloween_risa",
			Name:              "Halloween de risa",
			PromptAliases:     []string{"halloween risa", "terror de risa", "halloween comedia"},
			InclusionRules:    []string{"comedia", "terror", "parodia", "camp"},
			ExclusionRules:    []string{"terror extremo", "gore duro"},
			OrderingRules:     []string{"popularidad"},
			TemporaryByDefault: true,
		},
		{
			ID:                "navidad_tv",
			Name:              "Navidad TV",
			PromptAliases:     []string{"navidad tv", "pelis navideñas tv", "navidad familiar"},
			InclusionRules:    []string{"navidad", "aventura familiar", "feel-good", "clásicos TV"},
			ExclusionRules:    []string{"romance navideño genérico"},
			OrderingRules:     []string{"mezcla"},
			TemporaryByDefault: true,
		},
		{
			ID:                "gorilas",
			Name:              "Películas de gorilas",
			PromptAliases:     []string{"gorilas", "pelis de gorilas", "monos gigantes"},
			InclusionRules:    []string{"gorila", "king kong", "simios"},
			ExclusionRules:    []string{},
			OrderingRules:     []string{"año"},
			TemporaryByDefault: false,
		},
		{
			ID:                "nieve",
			Name:              "Películas de nieve",
			PromptAliases:     []string{"nieve", "películas de nieve", "como frozen", "invierno", "winter"},
			InclusionRules:    []string{"frozen", "snow", "winter", "ice", "navidad"},
			ExclusionRules:    []string{},
			OrderingRules:     []string{"mezcla"},
			TemporaryByDefault: false,
		},
	}
}
