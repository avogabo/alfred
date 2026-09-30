package geoffrey

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"alfred/internal/plex"
	"alfred/internal/tmdb"
)

type IdeaSuggestion struct {
	RecipeID        string       `json:"recipeId,omitempty"`
	RecipeName      string       `json:"recipeName,omitempty"`
	MatchedAliases  []string     `json:"matchedAliases,omitempty"`
	SuggestedTitles []plex.Video `json:"suggestedTitles"`
	SearchTerms     []string     `json:"searchTerms"`
}

type scoredVideo struct {
	item   plex.Video
	score  int
	reason string
}

// genreMap maps common Spanish/English words to TMDb Genre IDs.
var genreMap = map[string]int{
	"comedia":         35,
	"comedias":        35,
	"humor":           35,
	"divertida":       35,
	"divertidas":      35,
	"comedy":          35,
	"accion":          28,
	"acción":          28,
	"action":          28,
	"terror":          27,
	"miedo":           27,
	"horror":          27,
	"scary":           27,
	"animacion":       16,
	"animación":       16,
	"dibujos":         16,
	"animada":         16,
	"animadas":        16,
	"animation":       16,
	"familiar":        10751,
	"familiares":      10751,
	"familia":         10751,
	"infantil":        10751,
	"infantiles":      10751,
	"ninos":           10751,
	"niños":           10751,
	"kids":            10751,
	"family":          10751,
	"ciencia ficcion": 878,
	"ciencia ficción": 878,
	"scifi":           878,
	"sci-fi":          878,
	"drama":           18,
	"dramas":          18,
	"romance":         10749,
	"romantica":       10749,
	"romanticas":      10749,
	"romantico":       10749,
	"romanticos":      10749,
	"amor":            10749,
	"thriller":        53,
	"suspense":        53,
	"intriga":         53,
	"crimen":          80,
	"policiaca":       80,
	"policial":        80,
	"gangster":        80,
	"gangsters":       80,
	"mafia":           80,
	"crime":           80,
	"fantasia":        14,
	"fantasía":        14,
	"magia":           14,
	"fantasy":         14,
	"aventura":        12,
	"aventuras":       12,
	"adventure":       12,
	"belica":          10752,
	"bélica":          10752,
	"guerra":          10752,
	"war":             10752,
	"misterio":        9648,
	"mystery":         9648,
	"western":         37,
	"oeste":           37,
	"vaqueros":        37,
	"documental":      99,
	"documentales":    99,
	"documentary":     99,
}

// genrePlexNames maps TMDb Genre IDs to canonical Plex Genre tags.
var genrePlexNames = map[int][]string{
	35:    {"comedia", "comedy"},
	28:    {"accion", "acción", "action"},
	27:    {"terror", "horror"},
	16:    {"animacion", "animación", "animation"},
	10751: {"familiar", "family", "infantil", "kids"},
	878:   {"ciencia ficcion", "ciencia ficción", "sci-fi", "science fiction"},
	18:    {"drama"},
	10749: {"romance", "romantica", "romantic"},
	53:    {"thriller", "suspense"},
	80:    {"crimen", "crime"},
	14:    {"fantasia", "fantasía", "fantasy"},
	12:    {"aventura", "adventure"},
	10752: {"belica", "bélica", "guerra", "war"},
	9648:  {"misterio", "mystery"},
	37:    {"western"},
	99:    {"documental", "documentary"},
}

// thematicKeywordMap maps theme tokens to TMDb keyword IDs and search keywords.
var thematicKeywordMap = map[string][]int{
	"navidad":        {207317, 6513, 6514, 9799},
	"navidena":       {207317, 6513, 6514, 9799},
	"navidenas":      {207317, 6513, 6514, 9799},
	"navideno":       {207317, 6513, 6514, 9799},
	"navidenos":      {207317, 6513, 6514, 9799},
	"christmas":      {207317, 6513, 6514, 9799},
	"xmas":           {207317, 6513, 6514, 9799},
	"santa":          {6514, 207317},
	"claus":          {6514, 207317},
	"grinch":         {207317},
	"nieve":          {207317, 2636},
	"invierno":       {207317, 2636},
	"halloween":      {3335, 616, 2707},
	"brujas":         {616},
	"fantasma":       {2707},
	"fantasmas":      {2707},
	"zombie":         {12377, 186565},
	"zombies":        {12377, 186565},
	"zombis":         {12377, 186565},
	"superheroe":     {9715, 180547, 849},
	"superheroes":    {9715, 180547, 849},
	"vampiro":        {3133},
	"vampiros":       {3133},
	"robo":           {10051},
	"atracos":        {10051},
	"espia":          {470},
	"espias":         {470},
	"espionaje":      {470},
	"tiempo":         {4379},
	"viajes":         {4379},
	"apocalipsis":    {4458, 12371},
	"extraterrestre": {9951, 9882},
	"alien":          {9951},
	"espacio":        {9882, 14909},
	"espacial":       {9882, 14909},
	"espaciales":     {9882, 14909},

	// Cocina / Culinario / Gastronomía
	"cocina":       {1918, 18293, 222083, 238259, 291036, 178656},
	"cocinero":     {1918, 18293},
	"cocineros":    {1918, 18293},
	"cocinera":     {1918, 18293},
	"cocineras":    {1918, 18293},
	"cocinar":      {1918, 18293},
	"chef":         {18293, 1918},
	"chefs":        {18293, 1918},
	"repostero":    {238259, 291036, 18293, 1918},
	"reposteros":   {238259, 291036, 18293, 1918},
	"repostera":    {238259, 291036, 18293, 1918},
	"reposteras":   {238259, 291036, 18293, 1918},
	"reposteria":   {238259, 291036, 178656, 1918},
	"repostería":   {238259, 291036, 178656, 1918},
	"pastelero":    {238259, 291036, 178656},
	"pasteleros":   {238259, 291036, 178656},
	"pasteleria":   {238259, 291036, 178656},
	"pastelería":   {238259, 291036, 178656},
	"gastronomia":  {1918, 18293},
	"gastronomía":  {1918, 18293},
	"restaurante":  {18293, 1918},
	"restaurantes": {18293, 1918},

	// Animales / Mascotas
	"perro":    {15162, 208152, 8841},
	"perros":   {15162, 208152, 8841},
	"dalmata":  {158369, 15162},
	"dalmatas": {158369, 15162},
	"gato":     {977, 208152},
	"gatos":    {977, 208152},
	"mascota":  {208152, 15162, 977},
	"mascotas": {208152, 15162, 977},

	// Coches / Carreras
	"coche":         {310324, 286354, 830, 10039},
	"coches":        {286354, 310324, 830, 10039},
	"carreras":      {830, 10039},
	"carrera":       {830, 10039},
	"automovilismo": {830, 10039},
	"f1":            {830, 2383},

	// Ajedrez / Videojuegos
	"ajedrez":     {316},
	"videojuegos": {282},
	"gaming":      {282},

	// Música
	"musica":    {6021, 4344},
	"música":    {6021, 4344},
	"concierto": {4344},
	"banda":     {6021},
	"rock":      {11700, 6021},

	// Deporte
	"deporte":    {6075},
	"deportes":   {6075},
	"futbol":     {570},
	"fútbol":     {570},
	"baloncesto": {6078},
	"boxeo":      {2702},

	// Legal / Juicios
	"juicio":   {11038, 33519, 10909},
	"juicios":  {11038, 33519, 10909},
	"abogado":  {10909, 33519},
	"abogados": {10909, 33519},
	"tribunal": {33519, 11038},
}

var thematicSynonyms = map[string][]string{
	"navidad":     {"navidad", "christmas", "xmas", "noel", "holiday", "santa", "claus", "grinch", "klaus", "nochebuena", "reyes magos"},
	"navidena":    {"navidad", "christmas", "xmas", "noel", "holiday", "santa", "claus", "grinch", "klaus", "nochebuena", "reyes magos"},
	"navidenas":   {"navidad", "christmas", "xmas", "noel", "holiday", "santa", "claus", "grinch", "klaus", "nochebuena", "reyes magos"},
	"navideno":    {"navidad", "christmas", "xmas", "noel", "holiday", "santa", "claus", "grinch", "klaus", "nochebuena", "reyes magos"},
	"navidenos":   {"navidad", "christmas", "xmas", "noel", "holiday", "santa", "claus", "grinch", "klaus", "nochebuena", "reyes magos"},
	"christmas":   {"navidad", "christmas", "xmas", "noel", "holiday", "santa", "claus", "grinch", "klaus"},
	"nieve":       {"nieve", "snow", "winter", "ice", "frozen"},
	"invierno":    {"invierno", "winter", "snow", "ice"},
	"halloween":   {"halloween", "bruja", "scary", "fantasma", "ghost", "monstruo"},
	"zombie":      {"zombie", "zombies", "zombis", "walking dead", "infectados"},
	"superheroe":  {"superheroe", "superhero", "marvel", "dc", "avengers", "batman", "superman", "spiderman"},
	"espacio":     {"espacio", "space", "galaxia", "galaxy", "alien", "extraterrestre", "interestelar", "interstellar", "star"},
	"espacial":    {"espacio", "space", "galaxia", "galaxy", "alien", "extraterrestre", "interestelar", "interstellar", "star"},
	"cocina":      {"cocina", "cocinero", "cocineros", "cocinera", "cocineras", "cocinar", "chef", "chefs", "repostero", "reposteros", "repostera", "reposteras", "reposteria", "repostería", "pastelero", "pasteleros", "pastelera", "pasteleras", "pasteleria", "pastelería", "gastronomia", "gastronomía", "restaurante", "restaurantes", "culinario", "culinaria", "receta", "recetas", "gourmet", "baking", "pastry"},
	"cocinero":    {"cocina", "cocinero", "cocineros", "cocinera", "cocinar", "chef", "chefs", "repostero", "reposteros", "reposteria", "pastelero", "pasteleros", "pasteleria", "gastronomia", "restaurante"},
	"cocineros":   {"cocina", "cocinero", "cocineros", "cocinera", "cocinar", "chef", "chefs", "repostero", "reposteros", "reposteria", "pastelero", "pasteleros", "pasteleria", "gastronomia", "restaurante"},
	"chef":        {"chef", "chefs", "cocina", "cocinero", "cocineros", "repostero", "reposteros", "reposteria", "pastelero", "pasteleros", "pasteleria", "gastronomia", "gourmet", "restaurante"},
	"chefs":       {"chef", "chefs", "cocina", "cocinero", "cocineros", "repostero", "reposteros", "reposteria", "pastelero", "pasteleros", "pasteleria", "gastronomia", "gourmet", "restaurante"},
	"repostero":   {"repostero", "reposteros", "repostera", "reposteras", "reposteria", "repostería", "pastelero", "pasteleros", "pasteleria", "pastelería", "chef", "chefs", "cocina", "cocinero", "cocineros", "dulces"},
	"reposteros":  {"repostero", "reposteros", "repostera", "reposteras", "reposteria", "repostería", "pastelero", "pasteleros", "pasteleria", "pastelería", "chef", "chefs", "cocina", "cocinero", "cocineros", "dulces"},
	"reposteria":  {"repostero", "reposteros", "repostera", "reposteras", "reposteria", "repostería", "pastelero", "pasteleros", "pasteleria", "pastelería", "chef", "chefs", "cocina", "cocinero", "cocineros", "dulces"},
	"repostería":  {"repostero", "reposteros", "repostera", "reposteras", "reposteria", "repostería", "pastelero", "pasteleros", "pasteleria", "pastelería", "chef", "chefs", "cocina", "cocinero", "cocineros", "dulces"},
	"pastelero":   {"pastelero", "pasteleros", "pasteleria", "pastelería", "repostero", "reposteros", "reposteria", "repostería", "chef", "chefs", "cocina", "cocinero"},
	"pasteleros":  {"pastelero", "pasteleros", "pasteleria", "pastelería", "repostero", "reposteros", "reposteria", "repostería", "chef", "chefs", "cocina", "cocinero"},
	"pasteleria":  {"pastelero", "pasteleros", "pasteleria", "pastelería", "repostero", "reposteros", "reposteria", "repostería", "chef", "chefs", "cocina", "cocinero"},
	"pastelería":  {"pastelero", "pasteleros", "pasteleria", "pastelería", "repostero", "reposteros", "reposteria", "repostería", "chef", "chefs", "cocina", "cocinero"},
	"gastronomia": {"gastronomia", "gastronomía", "cocina", "cocinero", "cocineros", "chef", "chefs", "restaurante", "culinario", "gourmet"},
	"gastronomía": {"gastronomia", "gastronomía", "cocina", "cocinero", "cocineros", "chef", "chefs", "restaurante", "culinario", "gourmet"},
	"restaurante": {"restaurante", "restaurantes", "cocina", "chef", "cocinero", "gastronomia"},
	"perro":       {"perro", "perros", "perrito", "perritos", "dalmata", "dalmatas", "can", "canino", "mascota", "mascotas", "dog"},
	"perros":      {"perro", "perros", "perrito", "perritos", "dalmata", "dalmatas", "can", "canino", "mascota", "mascotas", "dog"},
	"dalmata":     {"dalmata", "dalmatas", "perro", "perros", "mascota"},
	"dalmatas":    {"dalmata", "dalmatas", "perro", "perros", "mascota"},
	"gato":        {"gato", "gatos", "gatito", "gatitos", "felino", "mascota", "mascotas", "cat"},
	"gatos":       {"gato", "gatos", "gatito", "gatitos", "felino", "mascota", "mascotas", "cat"},
	"coche":       {"coche", "coches", "auto", "autos", "automovil", "automoviles", "carrera", "carreras", "car", "cars"},
	"coches":      {"coche", "coches", "auto", "autos", "automovil", "automoviles", "carrera", "carreras", "car", "cars"},
	"carreras":    {"carrera", "carreras", "coche", "coches", "auto", "autos", "velocidad", "f1", "racing"},
	"ajedrez":     {"ajedrez", "chess", "jaque"},
}

// keywordTranslations provides English equivalents for TMDb keyword searches.
var keywordTranslations = map[string][]string{
	"cocina":         {"cooking", "chef"},
	"cocinero":       {"chef", "cook"},
	"chef":           {"chef", "cooking"},
	"repostero":      {"pastry chef", "pastry"},
	"reposteria":     {"pastry", "baking"},
	"pastelero":      {"pastry chef"},
	"pasteleria":     {"pastry", "baking"},
	"gastronomia":    {"culinary", "cooking"},
	"restaurante":    {"restaurant"},
	"perro":          {"dog"},
	"perros":         {"dog"},
	"gato":           {"cat"},
	"gatos":          {"cat"},
	"mascota":        {"pet"},
	"mascotas":       {"pet"},
	"ajedrez":        {"chess"},
	"carreras":       {"car race", "racing"},
	"coches":         {"cars"},
	"musica":         {"music"},
	"musical":        {"musical"},
	"deporte":        {"sports"},
	"deportes":       {"sports"},
	"futbol":         {"soccer", "football"},
	"baloncesto":     {"basketball"},
	"boxeo":          {"boxing"},
	"juicio":         {"courtroom", "lawyer"},
	"abogados":       {"lawyer"},
	"mafia":          {"mafia", "gangster"},
	"superheroe":     {"superhero"},
	"superheroes":    {"superhero"},
	"navidad":        {"christmas"},
	"halloween":      {"halloween"},
	"zombie":         {"zombie"},
	"zombies":        {"zombie"},
	"vampiro":        {"vampire"},
	"vampiros":       {"vampire"},
	"espia":          {"spy", "espionage"},
	"espias":         {"spy", "espionage"},
	"apocalipsis":    {"apocalypse"},
	"alien":          {"alien"},
	"extraterrestre": {"extraterrestrial"},
	"espacio":        {"space", "outer space"},
}

type ExtractedFacets struct {
	GenreIDs        []int
	ThematicKWIDs   []int
	ThematicWords   []string
	YearGte         int
	YearLte         int
	ExactYear       int
	RawTokens       []string
	MatchedRecipeID string
	MatchedRecipeName string
	MatchedAliases  []string
}

func (a *App) SuggestFromIdea(sectionKey, idea string) (IdeaSuggestion, error) {
	facets := a.extractFacets(idea)
	suggestion := IdeaSuggestion{
		RecipeID:       facets.MatchedRecipeID,
		RecipeName:     facets.MatchedRecipeName,
		MatchedAliases: facets.MatchedAliases,
	}

	seenTerms := map[string]bool{}
	for _, w := range facets.ThematicWords {
		appendIdeaTerm(w, seenTerms, &suggestion.SearchTerms)
	}
	for _, tok := range facets.RawTokens {
		appendIdeaTerm(tok, seenTerms, &suggestion.SearchTerms)
	}
	if len(suggestion.SearchTerms) == 0 {
		suggestion.SearchTerms = []string{idea}
	}

	// 1. External Discovery with TMDb
	tmdbByID := make(map[int]tmdb.Movie)
	tmdbByTitle := make(map[string]int)

	if a.tmdb != nil && a.tmdb.Enabled() {
		// Only run DiscoverMovie if we actually have specific filters (genre, keyword, or year constraint)!
		// If none are specified, DiscoverMovie without filters returns generic all-time blockbusters (e.g. Avengers, Titanic, Avatar).
		hasFilters := len(facets.GenreIDs) > 0 || len(facets.ThematicKWIDs) > 0 || facets.ExactYear > 0 || facets.YearGte > 0
		var movies []tmdb.Movie
		if hasFilters {
			opts := tmdb.DiscoverMovieOptions{
				WithGenres:   facets.GenreIDs,
				WithKeywords: facets.ThematicKWIDs,
				YearGte:      facets.YearGte,
				YearLte:      facets.YearLte,
				Year:         facets.ExactYear,
				SortBy:       "vote_count.desc",
				Language:     "es-ES",
			}
			var err error
			movies, err = a.tmdb.DiscoverMovie(opts)
			if err == nil && len(movies) > 0 {
				for _, m := range movies {
					tmdbByID[m.ID] = m
					registerTMDbTitle(m, tmdbByTitle)
				}
			}
		}

		// If discover had no keywords or few results, search with keywords or seed tokens
		if len(movies) < 5 {
			for _, term := range suggestion.SearchTerms {
				termsToSearch := []string{term}
				if translations, ok := keywordTranslations[term]; ok {
					termsToSearch = append(termsToSearch, translations...)
				}
				for _, kwTerm := range termsToSearch {
					if kwRes, err := a.tmdb.SearchKeyword(kwTerm); err == nil && len(kwRes) > 0 {
						opts2 := tmdb.DiscoverMovieOptions{
							WithGenres:   facets.GenreIDs,
							WithKeywords: []int{kwRes[0].ID},
							YearGte:      facets.YearGte,
							YearLte:      facets.YearLte,
							SortBy:       "vote_count.desc",
							Language:     "es-ES",
						}
						if extra, err := a.tmdb.DiscoverMovie(opts2); err == nil {
							for _, m := range extra {
								tmdbByID[m.ID] = m
								registerTMDbTitle(m, tmdbByTitle)
							}
						}
					}
				}
				if searchMovies, err := a.tmdb.SearchMovie(term); err == nil {
					for _, m := range searchMovies {
						tmdbByID[m.ID] = m
						registerTMDbTitle(m, tmdbByTitle)
					}
				}
				if len(tmdbByID) >= 40 {
					break
				}
			}
		}
	}

	// 2. Scan User's Plex Library
	var libraryVideos []plex.Video
	if a.plex != nil && a.plex.Configured() {
		if vids, err := a.plex.ListLibraryVideos(sectionKey); err == nil && len(vids) > 0 {
			libraryVideos = vids
		}
	}

	scoredMap := make(map[string]*scoredVideo)

	if len(libraryVideos) > 0 {
		// Hybrid Intelligent Matching against full library
		for _, vid := range libraryVideos {
			score, reason := scoreLibraryVideo(vid, facets, tmdbByID, tmdbByTitle)
			if score >= 60 {
				vid.Score = score
				vid.MatchReason = reason
				scoredMap[vid.RatingKey] = &scoredVideo{item: vid, score: score, reason: reason}
			}
		}
	} else {
		// Fallback: If library listing is not available, use individual Plex search queries
		for idx, term := range suggestion.SearchTerms {
			results, err := a.Search(sectionKey, term)
			if err != nil {
				continue
			}
			for _, item := range results {
				if item.RatingKey == "" {
					continue
				}
				score := fallbackScoreCandidate(item, facets.RawTokens, term, idx)
				if current, ok := scoredMap[item.RatingKey]; !ok || score > current.score {
					scoredMap[item.RatingKey] = &scoredVideo{item: item, score: score, reason: "plex_text_search"}
				}
			}
		}
	}

	// 3. Sort by score DESC, then Year DESC, then Title ASC
	list := make([]scoredVideo, 0, len(scoredMap))
	for _, item := range scoredMap {
		if item.score > 0 {
			list = append(list, *item)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].score != list[j].score {
			return list[i].score > list[j].score
		}
		if list[i].item.Year != list[j].item.Year {
			return list[i].item.Year > list[j].item.Year
		}
		return strings.ToLower(list[i].item.Title) < strings.ToLower(list[j].item.Title)
	})

	for _, item := range list {
		suggestion.SuggestedTitles = append(suggestion.SuggestedTitles, item.item)
	}
	if len(suggestion.SuggestedTitles) > 24 {
		suggestion.SuggestedTitles = suggestion.SuggestedTitles[:24]
	}

	if suggestion.RecipeName == "" && len(facets.RawTokens) > 0 {
		suggestion.RecipeName = strings.Title(idea)
	}

	return suggestion, nil
}

func registerTMDbTitle(m tmdb.Movie, titleMap map[string]int) {
	year := releaseYear(m.Release)
	if m.Title != "" {
		k := fmt.Sprintf("%s:%d", normalizeIdea(m.Title), year)
		titleMap[k] = m.ID
	}
	if m.OriginalTitle != "" {
		k := fmt.Sprintf("%s:%d", normalizeIdea(m.OriginalTitle), year)
		titleMap[k] = m.ID
	}
}

func releaseYear(rel string) int {
	if len(rel) >= 4 {
		if y, err := strconv.Atoi(rel[:4]); err == nil {
			return y
		}
	}
	return 0
}

func scoreLibraryVideo(
	vid plex.Video,
	facets ExtractedFacets,
	tmdbByID map[int]tmdb.Movie,
	tmdbByTitle map[string]int,
) (int, string) {
	normTitle := normalizeIdea(vid.Title)
	normOrig := normalizeIdea(vid.OriginalTitle)
	normSummary := normalizeIdea(vid.Summary)

	// Check Year/Decade Constraint
	yearPenalty := 0
	if facets.YearGte > 0 && vid.Year > 0 {
		if vid.Year < facets.YearGte || (facets.YearLte > 0 && vid.Year > facets.YearLte) {
			yearPenalty = -35
		}
	} else if facets.ExactYear > 0 && vid.Year > 0 {
		if int(math.Abs(float64(vid.Year-facets.ExactYear))) > 1 {
			yearPenalty = -35
		}
	}

	// 1. Direct TMDb ID match from Curated TMDb Discover
	if vid.TMDBID > 0 {
		if _, ok := tmdbByID[vid.TMDBID]; ok {
			return 100 + yearPenalty, "tmdb_curated_id"
		}
	}

	// 2. Direct Title + Year match from Curated TMDb Discover
	keyA := fmt.Sprintf("%s:%d", normTitle, vid.Year)
	keyB := fmt.Sprintf("%s:%d", normOrig, vid.Year)
	if _, ok := tmdbByTitle[keyA]; ok {
		return 95 + yearPenalty, "tmdb_curated_title"
	}
	if normOrig != "" {
		if _, ok := tmdbByTitle[keyB]; ok {
			return 95 + yearPenalty, "tmdb_curated_original_title"
		}
	}

	// 3. Local Library Match: Genre match + Thematic word in Summary or Title
	hasTargetGenre := false
	if len(facets.GenreIDs) > 0 {
		for _, gid := range facets.GenreIDs {
			names := genrePlexNames[gid]
			for _, g := range vid.Genres {
				normG := normalizeIdea(g)
				for _, name := range names {
					if strings.Contains(normG, name) || strings.Contains(name, normG) {
						hasTargetGenre = true
						break
					}
				}
				if hasTargetGenre {
					break
				}
			}
			if hasTargetGenre {
				break
			}
		}
	}

	hasThematicKeywordInTitle := false
	hasThematicKeywordInSummary := false
	for _, w := range facets.ThematicWords {
		if w == "" {
			continue
		}
		if containsWord(normTitle, w) {
			hasThematicKeywordInTitle = true
		}
		if containsWord(normSummary, w) {
			hasThematicKeywordInSummary = true
		}
		for _, syn := range thematicSynonyms[w] {
			if containsWord(normTitle, syn) {
				hasThematicKeywordInTitle = true
			}
			if containsWord(normSummary, syn) {
				hasThematicKeywordInSummary = true
			}
		}
	}

	if hasTargetGenre && hasThematicKeywordInTitle {
		return 90 + yearPenalty, "library_genre_and_title_thematic_match"
	}

	if hasTargetGenre && hasThematicKeywordInSummary {
		return 85 + yearPenalty, "library_genre_thematic_match"
	}

	if hasThematicKeywordInTitle {
		return 80 + yearPenalty, "library_title_thematic_match"
	}

	if hasThematicKeywordInSummary {
		return 75 + yearPenalty, "library_summary_thematic_match"
	}

	// If prompt only requested genres and this title matches target genre
	if len(facets.ThematicWords) == 0 && hasTargetGenre {
		return 70 + yearPenalty, "library_genre_match"
	}

	// If prompt only requested decade or year and this title matches
	if len(facets.GenreIDs) == 0 && len(facets.ThematicWords) == 0 {
		if facets.YearGte > 0 && vid.Year >= facets.YearGte && (facets.YearLte == 0 || vid.Year <= facets.YearLte) {
			return 70, "library_year_match"
		}
		if facets.ExactYear > 0 && vid.Year == facets.ExactYear {
			return 70, "library_year_match"
		}
	}

	return 0, ""
}

func containsWord(text, word string) bool {
	if text == "" || word == "" {
		return false
	}
	pattern := fmt.Sprintf(`(?i)\b%s\b`, regexp.QuoteMeta(word))
	matched, _ := regexp.MatchString(pattern, text)
	return matched
}

func (a *App) extractFacets(idea string) ExtractedFacets {
	norm := normalizeIdea(idea)
	tokens := tokenizeIdea(norm)
	facets := ExtractedFacets{
		RawTokens: tokens,
	}

	// 1. Recipe alias matching
	for _, recipe := range a.Recipes() {
		for _, alias := range recipe.PromptAliases {
			aliasNorm := normalizeIdea(alias)
			if aliasNorm != "" && (strings.Contains(norm, aliasNorm) || strings.Contains(aliasNorm, norm)) {
				facets.MatchedRecipeID = recipe.ID
				facets.MatchedRecipeName = recipe.Name
				facets.MatchedAliases = append(facets.MatchedAliases, alias)
				for _, r := range recipe.InclusionRules {
					facets.ThematicWords = append(facets.ThematicWords, normalizeIdea(r))
				}
				break
			}
		}
		if facets.MatchedRecipeID != "" {
			break
		}
	}

	// 2. Decade / Year extraction
	reDecade := regexp.MustCompile(`(?i)(?:anos|años|los)\s*(\d{2})s?|(19\d{2}|20\d{2})s?`)
	matches := reDecade.FindAllStringSubmatch(norm, -1)
	for _, m := range matches {
		if m[1] != "" {
			// e.g. "80", "90"
			d, _ := strconv.Atoi(m[1])
			if d >= 20 && d <= 99 {
				facets.YearGte = 1900 + d
				facets.YearLte = 1900 + d + 9
			} else if d >= 0 && d <= 25 {
				facets.YearGte = 2000 + d
				facets.YearLte = 2000 + d + 9
			}
		} else if m[2] != "" {
			// e.g. "1980", "1994"
			y, _ := strconv.Atoi(m[2])
			if strings.HasSuffix(m[0], "s") || strings.HasSuffix(m[0], "0") {
				facets.YearGte = y
				facets.YearLte = y + 9
			} else {
				facets.ExactYear = y
			}
		}
	}

	// 3. Genre extraction
	seenGenres := make(map[int]bool)
	for _, tok := range tokens {
		if gid, ok := genreMap[tok]; ok {
			if !seenGenres[gid] {
				seenGenres[gid] = true
				facets.GenreIDs = append(facets.GenreIDs, gid)
			}
		}
	}
	// Check multi-word genre phrases
	for phrase, gid := range genreMap {
		if strings.Contains(norm, phrase) && !seenGenres[gid] {
			seenGenres[gid] = true
			facets.GenreIDs = append(facets.GenreIDs, gid)
		}
	}

	// 4. Thematic keyword extraction
	seenKW := make(map[int]bool)
	seenWords := make(map[string]bool)
	for _, tok := range tokens {
		if _, isGenre := genreMap[tok]; isGenre {
			continue
		}
		if kwIDs, ok := thematicKeywordMap[tok]; ok {
			for _, kw := range kwIDs {
				if !seenKW[kw] {
					seenKW[kw] = true
					facets.ThematicKWIDs = append(facets.ThematicKWIDs, kw)
				}
			}
			if !seenWords[tok] {
				seenWords[tok] = true
				facets.ThematicWords = append(facets.ThematicWords, tok)
			}
		} else if len(tok) >= 4 && !isStopWord(tok) {
			if !seenWords[tok] {
				seenWords[tok] = true
				facets.ThematicWords = append(facets.ThematicWords, tok)
			}
		}
	}

	return facets
}

func isStopWord(s string) bool {
	switch s {
	case "para", "como", "pero", "sobre", "entre", "este", "esta", "estos", "estas", "todos", "todas", "peliculas", "pelicula", "peli", "pelis", "series", "serie", "cine", "film", "films", "movie", "movies", "de", "del", "la", "el", "los", "las", "un", "una", "unos", "unas", "con", "sin", "por", "que", "mas", "muy", "mi", "mis", "tu", "tus", "su", "sus":
		return true
	default:
		return false
	}
}

func fallbackScoreCandidate(item plex.Video, ideaTokens []string, term string, termIndex int) int {
	title := normalizeIdea(item.Title)
	score := 0
	matched := false
	if strings.Contains(title, term) {
		score += 40
		matched = true
	}
	for _, token := range ideaTokens {
		if token == "" {
			continue
		}
		if strings.Contains(title, token) {
			score += 30
			matched = true
		}
	}
	if !matched {
		return 0
	}
	if item.Year >= 1990 {
		score += 4
	}
	if item.Type == "movie" {
		score += 4
	}
	return score
}

func appendIdeaTerm(term string, seen map[string]bool, out *[]string) {
	if term == "" || seen[term] {
		return
	}
	seen[term] = true
	*out = append(*out, term)
}

func normalizeIdea(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	replacer := strings.NewReplacer(
		",", " ", ".", " ", ":", " ", ";", " ", "-", " ", "_", " ", "/", " ",
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	)
	return strings.Join(strings.Fields(replacer.Replace(in)), " ")
}

func tokenizeIdea(idea string) []string {
	idea = normalizeIdea(idea)
	parts := strings.Fields(idea)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) >= 3 {
			out = append(out, part)
		}
	}
	return out
}
