package plex

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SearchResponse struct {
	Videos []Video `xml:"Video"`
	Dirs   []Video `xml:"Directory"`
}

type Video struct {
	RatingKey string `xml:"ratingKey,attr" json:"ratingKey"`
	Title     string `xml:"title,attr" json:"title"`
	Type      string `xml:"type,attr" json:"type"`
	Year      int    `xml:"year,attr" json:"year"`
	Thumb     string `xml:"thumb,attr" json:"thumb"`
	Art       string `xml:"art,attr" json:"art"`
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
