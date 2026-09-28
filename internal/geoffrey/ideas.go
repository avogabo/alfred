package geoffrey

import (
	"sort"
	"strings"

	"alfred/internal/plex"
)

type IdeaSuggestion struct {
	RecipeID        string       `json:"recipeId,omitempty"`
	RecipeName      string       `json:"recipeName,omitempty"`
	MatchedAliases  []string     `json:"matchedAliases,omitempty"`
	SuggestedTitles []plex.Video `json:"suggestedTitles"`
	SearchTerms     []string     `json:"searchTerms"`
}

type scoredVideo struct {
	item  plex.Video
	score int
}

var ideaSynonyms = map[string][]string{
	"nieve":      {"snow", "winter", "ice", "frozen", "christmas"},
	"invierno":   {"winter", "snow", "ice", "frozen"},
	"navidad":    {"christmas", "noel", "holiday", "xmas"},
	"frozen":     {"snow", "ice", "winter", "elsa", "anna"},
	"gorila":     {"gorilla", "kong", "ape", "simio"},
	"gorilas":    {"gorilla", "kong", "ape", "simio"},
	"halloween":  {"scary", "monster", "ghost", "horror"},
	"comedia":    {"fun", "parody", "laugh", "family"},
	"familiar":   {"family", "kids", "animation"},
	"polar":      {"ice", "snow", "winter", "north"},
	"helada":     {"ice", "snow", "winter", "frozen"},
	"frío":       {"cold", "ice", "snow", "winter"},
}

func (a *App) SuggestFromIdea(sectionKey, idea string) (IdeaSuggestion, error) {
	idea = normalizeIdea(idea)
	recipes := a.Recipes()
	suggestion := IdeaSuggestion{}
	seenTerms := map[string]bool{}
	ideaTokens := tokenizeIdea(idea)
	for _, recipe := range recipes {
		var matched []string
		for _, alias := range recipe.PromptAliases {
			aliasNorm := normalizeIdea(alias)
			if aliasNorm == "" {
				continue
			}
			if strings.Contains(idea, aliasNorm) || strings.Contains(aliasNorm, idea) {
				matched = append(matched, alias)
			}
		}
		if len(matched) > 0 {
			suggestion.RecipeID = recipe.ID
			suggestion.RecipeName = recipe.Name
			suggestion.MatchedAliases = matched
			for _, term := range recipe.InclusionRules {
				appendIdeaTerm(cleanIdeaTerm(term), seenTerms, &suggestion.SearchTerms)
			}
			break
		}
	}
	for _, token := range ideaTokens {
		appendIdeaTerm(token, seenTerms, &suggestion.SearchTerms)
		for _, syn := range ideaSynonyms[token] {
			appendIdeaTerm(cleanIdeaTerm(syn), seenTerms, &suggestion.SearchTerms)
		}
	}
	if len(suggestion.SearchTerms) == 0 {
		suggestion.SearchTerms = []string{idea}
	}
	if a.tmdb != nil && a.tmdb.Enabled() {
		for _, seed := range append([]string{idea}, suggestion.SearchTerms...) {
			movies, err := a.tmdb.SearchMovie(seed)
			if err != nil {
				continue
			}
			for _, movie := range movies {
				appendIdeaTerm(cleanIdeaTerm(movie.Title), seenTerms, &suggestion.SearchTerms)
			}
			if len(movies) > 0 {
				break
			}
		}
	}

	scored := map[string]*scoredVideo{}
	for idx, term := range suggestion.SearchTerms {
		results, err := a.Search(sectionKey, term)
		if err != nil {
			return suggestion, err
		}
		for _, item := range results {
			if item.RatingKey == "" {
				continue
			}
			score := scoreCandidate(item, ideaTokens, term, idx)
			if current, ok := scored[item.RatingKey]; !ok || score > current.score {
				scored[item.RatingKey] = &scoredVideo{item: item, score: score}
			}
		}
	}
	list := make([]scoredVideo, 0, len(scored))
	for _, item := range scored {
		if item.score <= 0 {
			continue
		}
		list = append(list, *item)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].score == list[j].score {
			return strings.ToLower(list[i].item.Title) < strings.ToLower(list[j].item.Title)
		}
		return list[i].score > list[j].score
	})
	for _, item := range list {
		suggestion.SuggestedTitles = append(suggestion.SuggestedTitles, item.item)
	}
	if len(suggestion.SuggestedTitles) > 18 {
		suggestion.SuggestedTitles = suggestion.SuggestedTitles[:18]
	}
	if suggestion.RecipeName == "" && len(ideaTokens) > 0 {
		suggestion.RecipeName = strings.Title(idea)
	}
	return suggestion, nil
}

func appendIdeaTerm(term string, seen map[string]bool, out *[]string) {
	if term == "" || seen[term] {
		return
	}
	seen[term] = true
	*out = append(*out, term)
}

func scoreCandidate(item plex.Video, ideaTokens []string, term string, termIndex int) int {
	title := normalizeIdea(item.Title)
	score := 0
	if termIndex == 0 {
		score += 20
	}
	if strings.Contains(title, term) {
		score += 30
	}
	for _, token := range ideaTokens {
		if token == "" {
			continue
		}
		if strings.Contains(title, token) {
			score += 18
		}
		for _, syn := range ideaSynonyms[token] {
			syn = cleanIdeaTerm(syn)
			if syn != "" && strings.Contains(title, syn) {
				score += 12
			}
		}
	}
	if item.Year >= 1990 {
		score += 4
	}
	if item.Type == "movie" {
		score += 4
	}
	return score
}

func normalizeIdea(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	replacer := strings.NewReplacer(",", " ", ".", " ", ":", " ", ";", " ", "-", " ", "_", " ", "/", " ", "á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u")
	return strings.Join(strings.Fields(replacer.Replace(in)), " ")
}

func cleanIdeaTerm(in string) string {
	return normalizeIdea(in)
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
