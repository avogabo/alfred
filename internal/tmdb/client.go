package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	apiKey string
	http   *http.Client
}

type SearchMovieResponse struct {
	Results []Movie `json:"results"`
}

type Movie struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Name          string  `json:"name"`
	Release       string  `json:"release_date"`
	FirstAir      string  `json:"first_air_date"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	GenreIDs      []int   `json:"genre_ids"`
	VoteCount     int     `json:"vote_count"`
	VoteAverage   float64 `json:"vote_average"`
	Popularity    float64 `json:"popularity"`
}

type Keyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type KeywordSearchResponse struct {
	Results []Keyword `json:"results"`
}

type Collection struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	PosterPath   string `json:"poster_path"`
	BackdropPath string `json:"backdrop_path"`
}

type CollectionSearchResponse struct {
	Results []Collection `json:"results"`
}

type DiscoverMovieOptions struct {
	WithGenres   []int
	WithKeywords []int
	SortBy       string
	Year         int
	YearGte      int
	YearLte      int
	Language     string
	Page         int
}

func New(apiKey string) *Client {
	return &Client{apiKey: strings.TrimSpace(apiKey), http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) Enabled() bool { return c != nil && c.apiKey != "" }

func (c *Client) SearchMovie(query string) ([]Movie, error) {
	if !c.Enabled() || strings.TrimSpace(query) == "" {
		return nil, nil
	}
	u := "https://api.themoviedb.org/3/search/movie"
	q := url.Values{}
	q.Set("api_key", c.apiKey)
	q.Set("query", query)
	q.Set("include_adult", "false")
	q.Set("language", "es-ES")
	resp, err := c.http.Get(u + "?" + q.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tmdb search failed: %s", resp.Status)
	}
	var out SearchMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Results, nil
}

func (c *Client) SearchMulti(query string) ([]Movie, error) {
	if !c.Enabled() || strings.TrimSpace(query) == "" {
		return nil, nil
	}
	u := "https://api.themoviedb.org/3/search/multi"
	q := url.Values{}
	q.Set("api_key", c.apiKey)
	q.Set("query", query)
	q.Set("language", "es-ES")
	q.Set("page", "1")
	resp, err := c.http.Get(u + "?" + q.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tmdb multi search failed: %s", resp.Status)
	}
	var out SearchMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	for i := range out.Results {
		if out.Results[i].Title == "" && out.Results[i].Name != "" {
			out.Results[i].Title = out.Results[i].Name
		}
		if out.Results[i].Release == "" && out.Results[i].FirstAir != "" {
			out.Results[i].Release = out.Results[i].FirstAir
		}
	}
	return out.Results, nil
}

func (c *Client) SearchCollection(query string) ([]Collection, error) {
	if !c.Enabled() || strings.TrimSpace(query) == "" {
		return nil, nil
	}
	u := "https://api.themoviedb.org/3/search/collection"
	q := url.Values{}
	q.Set("api_key", c.apiKey)
	q.Set("query", query)
	q.Set("language", "es-ES")
	q.Set("page", "1")
	resp, err := c.http.Get(u + "?" + q.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tmdb collection search failed: %s", resp.Status)
	}
	var out CollectionSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Results, nil
}


func (c *Client) SearchKeyword(query string) ([]Keyword, error) {
	if !c.Enabled() || strings.TrimSpace(query) == "" {
		return nil, nil
	}
	u := "https://api.themoviedb.org/3/search/keyword"
	q := url.Values{}
	q.Set("api_key", c.apiKey)
	q.Set("query", query)
	q.Set("page", "1")
	resp, err := c.http.Get(u + "?" + q.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tmdb keyword search failed: %s", resp.Status)
	}
	var out KeywordSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Results, nil
}

func (c *Client) DiscoverMovie(opts DiscoverMovieOptions) ([]Movie, error) {
	if !c.Enabled() {
		return nil, nil
	}
	u := "https://api.themoviedb.org/3/discover/movie"
	q := url.Values{}
	q.Set("api_key", c.apiKey)
	q.Set("include_adult", "false")
	if opts.Language != "" {
		q.Set("language", opts.Language)
	} else {
		q.Set("language", "es-ES")
	}
	if opts.SortBy != "" {
		q.Set("sort_by", opts.SortBy)
	} else {
		q.Set("sort_by", "vote_count.desc")
	}

	if len(opts.WithGenres) > 0 {
		var genres []string
		for _, g := range opts.WithGenres {
			if g > 0 {
				genres = append(genres, fmt.Sprintf("%d", g))
			}
		}
		if len(genres) > 0 {
			q.Set("with_genres", strings.Join(genres, ","))
		}
	}

	if len(opts.WithKeywords) > 0 {
		var kwList []string
		for _, k := range opts.WithKeywords {
			if k > 0 {
				kwList = append(kwList, fmt.Sprintf("%d", k))
			}
		}
		if len(kwList) > 0 {
			// Using pipe | means OR in TMDb keywords discover
			q.Set("with_keywords", strings.Join(kwList, "|"))
		}
	}

	if opts.Year > 0 {
		q.Set("primary_release_year", fmt.Sprintf("%d", opts.Year))
	}
	if opts.YearGte > 0 {
		q.Set("primary_release_date.gte", fmt.Sprintf("%d-01-01", opts.YearGte))
	}
	if opts.YearLte > 0 {
		q.Set("primary_release_date.lte", fmt.Sprintf("%d-12-31", opts.YearLte))
	}
	if opts.Page > 1 {
		q.Set("page", fmt.Sprintf("%d", opts.Page))
	} else {
		q.Set("page", "1")
	}

	resp, err := c.http.Get(u + "?" + q.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tmdb discover failed: %s", resp.Status)
	}
	var out SearchMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Results, nil
}
