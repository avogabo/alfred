package plex

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type libraryCacheEntry struct {
	items     []Video
	timestamp time.Time
}

type Client struct {
	baseURL     string
	token       string
	pathFrom    string
	pathTo      string
	http        *http.Client
	mu          sync.RWMutex
	machineID   string
	cacheMu     sync.RWMutex
	videoCache  map[string]libraryCacheEntry
}

type LibrariesResponse struct {
	Directories []Library `xml:"Directory"`
}

type Library struct {
	Key   string `xml:"key,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
}

type IdentityResponse struct {
	MachineIdentifier string `xml:"machineIdentifier,attr"`
	Version           string `xml:"version,attr"`
}

func New(baseURL, token, pathFrom, pathTo string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      strings.TrimSpace(token),
		pathFrom:   pathFrom,
		pathTo:     pathTo,
		http:       &http.Client{Timeout: 20 * time.Second},
		videoCache: make(map[string]libraryCacheEntry),
	}
}

func (c *Client) GetCachedVideos(sectionKey string) ([]Video, bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	if c.videoCache == nil {
		return nil, false
	}
	entry, ok := c.videoCache[sectionKey]
	if !ok || time.Since(entry.timestamp) > 3*time.Minute {
		return nil, false
	}
	return entry.items, true
}

func (c *Client) SetCachedVideos(sectionKey string, items []Video) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if c.videoCache == nil {
		c.videoCache = make(map[string]libraryCacheEntry)
	}
	c.videoCache[sectionKey] = libraryCacheEntry{
		items:     items,
		timestamp: time.Now(),
	}
}

func (c *Client) Configured() bool {
	return c.baseURL != "" && c.token != ""
}

func (c *Client) Libraries() ([]Library, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("plex client not configured")
	}
	u := c.baseURL + "/library/sections"
	q := url.Values{}
	q.Set("X-Plex-Token", c.token)
	u += "?" + q.Encode()
	resp, err := c.http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("plex libraries failed: %s", resp.Status)
	}
	var out LibrariesResponse
	if err := xml.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Directories, nil
}

func (c *Client) RefreshPath(targetPath string) error {
	if !c.Configured() {
		return nil
	}
	if targetPath == "" || targetPath == "." {
		return c.RefreshAll()
	}
	parent := filepath.Dir(targetPath)
	parent = c.translatePath(parent)
	if parent == "." || parent == "" || parent == "/" {
		return c.RefreshAll()
	}

	u := c.baseURL + "/library/sections/all/refresh"
	q := url.Values{}
	q.Set("path", parent)
	q.Set("X-Plex-Token", c.token)
	u += "?" + q.Encode()

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return c.RefreshAll()
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.RefreshAll()
	}
	log.Printf("alfred: plex refresh requested for %s", parent)
	return nil
}

func (c *Client) RefreshAll() error {
	if !c.Configured() {
		return nil
	}
	u := c.baseURL + "/library/sections/all/refresh"
	q := url.Values{}
	q.Set("X-Plex-Token", c.token)
	u += "?" + q.Encode()

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("plex refresh all failed: %s", resp.Status)
	}
	log.Printf("alfred: plex full library refresh triggered successfully")
	return nil
}

func (c *Client) translatePath(in string) string {
	from := strings.Trim(strings.TrimSpace(c.pathFrom), "\"'")
	to := strings.Trim(strings.TrimSpace(c.pathTo), "\"'")
	if from == "" || to == "" {
		return in
	}
	cleanIn := filepath.Clean(strings.Trim(strings.TrimSpace(in), "\"'"))
	cleanFrom := filepath.Clean(from)
	cleanTo := filepath.Clean(to)
	if cleanIn == cleanFrom {
		return cleanTo
	}
	prefix := cleanFrom + string(filepath.Separator)
	if strings.HasPrefix(cleanIn, prefix) {
		rest := strings.TrimPrefix(cleanIn, prefix)
		return filepath.Join(cleanTo, rest)
	}
	return in
}

func (c *Client) FetchImage(path string) (*http.Response, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("plex client not configured")
	}
	var u string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		u = path
	} else {
		u = c.baseURL + path
	}
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	u += sep + "X-Plex-Token=" + url.QueryEscape(c.token)
	return c.http.Get(u)
}

func (c *Client) MachineIdentifier() string {
	c.mu.RLock()
	if c.machineID != "" {
		defer c.mu.RUnlock()
		return c.machineID
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.machineID != "" {
		return c.machineID
	}

	if !c.Configured() {
		return "4d4119d5e3c654b31e3b3945315f92bc9dadffe2"
	}

	u := c.baseURL + "/identity?X-Plex-Token=" + url.QueryEscape(c.token)
	resp, err := c.http.Get(u)
	if err == nil {
		defer resp.Body.Close()
		var ident IdentityResponse
		if xml.NewDecoder(resp.Body).Decode(&ident) == nil && ident.MachineIdentifier != "" {
			c.machineID = ident.MachineIdentifier
			return c.machineID
		}
	}

	c.machineID = "4d4119d5e3c654b31e3b3945315f92bc9dadffe2"
	return c.machineID
}
