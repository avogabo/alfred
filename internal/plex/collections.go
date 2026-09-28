package plex

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type SearchResponse struct {
	Videos []Video `xml:"Video"`
	Dirs   []Video `xml:"Directory"`
}

type Video struct {
	RatingKey     string   `xml:"ratingKey,attr" json:"ratingKey"`
	Title         string   `xml:"title,attr" json:"title"`
	OriginalTitle string   `xml:"originalTitle,attr" json:"originalTitle,omitempty"`
	Type          string   `xml:"type,attr" json:"type"`
	Year          int      `xml:"year,attr" json:"year"`
	Thumb         string   `xml:"thumb,attr" json:"thumb"`
	Art           string   `xml:"art,attr" json:"art"`
	Summary       string   `xml:"summary,attr" json:"summary,omitempty"`
	GUID          string   `xml:"guid,attr" json:"guid,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	TMDBID        int      `json:"tmdbId,omitempty"`
	IMDBID        string   `json:"imdbId,omitempty"`
	Score         int      `json:"score,omitempty"`
	MatchReason   string   `json:"matchReason,omitempty"`
}

type rawMediaItem struct {
	RatingKey     string `xml:"ratingKey,attr"`
	Title         string `xml:"title,attr"`
	OriginalTitle string `xml:"originalTitle,attr"`
	Type          string `xml:"type,attr"`
	Year          int    `xml:"year,attr"`
	Thumb         string `xml:"thumb,attr"`
	Art           string `xml:"art,attr"`
	Summary       string `xml:"summary,attr"`
	GUID          string `xml:"guid,attr"`
	Genres        []struct {
		Tag string `xml:"tag,attr"`
	} `xml:"Genre"`
	Guids []struct {
		ID string `xml:"id,attr"`
	} `xml:"Guid"`
}

type LibraryItemsResponse struct {
	Videos []rawMediaItem `xml:"Video"`
	Dirs   []rawMediaItem `xml:"Directory"`
}

func (c *Client) ListLibraryVideos(sectionKey string) ([]Video, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("plex client not configured")
	}

	if cached, ok := c.GetCachedVideos(sectionKey); ok {
		return cached, nil
	}

	u := fmt.Sprintf("%s/library/sections/%s/all", c.baseURL, sectionKey)
	q := url.Values{}
	q.Set("X-Plex-Token", c.token)
	u += "?" + q.Encode()

	resp, err := c.http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("plex list library items failed: %s", resp.Status)
	}

	var out LibraryItemsResponse
	if err := xml.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	rawItems := append([]rawMediaItem{}, out.Videos...)
	rawItems = append(rawItems, out.Dirs...)

	items := make([]Video, 0, len(rawItems))
	for _, raw := range rawItems {
		v := Video{
			RatingKey:     raw.RatingKey,
			Title:         raw.Title,
			OriginalTitle: raw.OriginalTitle,
			Type:          raw.Type,
			Year:          raw.Year,
			Thumb:         raw.Thumb,
			Art:           raw.Art,
			Summary:       raw.Summary,
			GUID:          raw.GUID,
		}
		for _, g := range raw.Genres {
			tag := strings.TrimSpace(g.Tag)
			if tag != "" {
				v.Genres = append(v.Genres, tag)
			}
		}
		for _, guid := range raw.Guids {
			id := strings.TrimSpace(guid.ID)
			if strings.HasPrefix(id, "tmdb://") {
				if n, err := strconv.Atoi(strings.TrimPrefix(id, "tmdb://")); err == nil {
					v.TMDBID = n
				}
			} else if strings.HasPrefix(id, "imdb://") {
				v.IMDBID = strings.TrimPrefix(id, "imdb://")
			}
		}
		if v.TMDBID == 0 && strings.Contains(v.GUID, "themoviedb://") {
			parts := strings.Split(v.GUID, "themoviedb://")
			if len(parts) > 1 {
				numStr := strings.Split(parts[1], "?")[0]
				if n, err := strconv.Atoi(numStr); err == nil {
					v.TMDBID = n
				}
			}
		}
		items = append(items, v)
	}

	c.SetCachedVideos(sectionKey, items)
	return items, nil
}

type MetadataResponse struct {
	Directories []Collection `xml:"Directory"`
}

type Collection struct {
	RatingKey  string `xml:"ratingKey,attr" json:"ratingKey"`
	Title      string `xml:"title,attr" json:"title"`
	Type       string `xml:"type,attr" json:"type"`
	Subtype    string `xml:"subtype,attr" json:"subtype"`
	ChildCount int    `xml:"childCount,attr" json:"childCount"`
	Thumb      string `xml:"thumb,attr" json:"thumb"`
	Art        string `xml:"art,attr" json:"art"`
}

func (c *Client) Search(sectionKey, query string) ([]Video, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("plex client not configured")
	}
	u := fmt.Sprintf("%s/library/sections/%s/search", c.baseURL, sectionKey)
	q := url.Values{}
	q.Set("query", query)
	q.Set("X-Plex-Token", c.token)
	u += "?" + q.Encode()
	resp, err := c.http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("plex search failed: %s", resp.Status)
	}
	var out SearchResponse
	if err := xml.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	items := append([]Video{}, out.Videos...)
	items = append(items, out.Dirs...)
	return items, nil
}

func (c *Client) ListCollections(sectionKey string) ([]Collection, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("plex client not configured")
	}
	u := fmt.Sprintf("%s/library/sections/%s/collections", c.baseURL, sectionKey)
	q := url.Values{}
	q.Set("X-Plex-Token", c.token)
	u += "?" + q.Encode()
	resp, err := c.http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("plex list collections failed: %s", resp.Status)
	}
	var out MetadataResponse
	if err := xml.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Directories, nil
}

func (c *Client) CreateCollection(sectionKey, title string, ratingKeys []string) error {
	if !c.Configured() {
		return fmt.Errorf("plex client not configured")
	}
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("collection title is required")
	}
	if len(ratingKeys) == 0 {
		return fmt.Errorf("at least one rating key is required")
	}
	u := fmt.Sprintf("%s/library/collections", c.baseURL)
	q := url.Values{}
	q.Set("type", sectionTypeFromSectionKey(sectionKey))
	q.Set("title", title)
	q.Set("smart", "0")
	q.Set("sectionId", sectionKey)
	q.Set("uri", c.collectionURI(sectionKey, ratingKeys))
	q.Set("X-Plex-Token", c.token)
	u += "?" + q.Encode()
	req, err := http.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("plex create collection failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) DeleteCollection(ratingKey string) error {
	if !c.Configured() {
		return fmt.Errorf("plex client not configured")
	}
	u := fmt.Sprintf("%s/library/metadata/%s", c.baseURL, ratingKey)
	q := url.Values{}
	q.Set("X-Plex-Token", c.token)
	u += "?" + q.Encode()
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("plex delete collection failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) collectionURI(sectionKey string, ratingKeys []string) string {
	return fmt.Sprintf("server://%s/com.plexapp.plugins.library/library/metadata/%s", c.MachineIdentifier(), strings.Join(ratingKeys, ","))
}

func sectionTypeFromSectionKey(sectionKey string) string {
	if sectionKey == "2" {
		return "2"
	}
	return "1"
}
