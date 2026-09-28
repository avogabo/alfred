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
	Title      string `json:"title"`
	Name       string `json:"name"`
	Release    string `json:"release_date"`
	FirstAir   string `json:"first_air_date"`
	Overview   string `json:"overview"`
	PosterPath string `json:"poster_path"`
}

func New(apiKey string) *Client {
	return &Client{apiKey: strings.TrimSpace(apiKey), http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) Enabled() bool { return c != nil && c.apiKey != "" }

func (c *Client) SearchMovie(query string) ([]Movie, error) {
	if !c.Enabled() {
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
