package plex

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) SetPosterURL(ratingKey, posterURL string) error {
	if !c.Configured() {
		return fmt.Errorf("plex client not configured")
	}
	posterURL = strings.TrimSpace(posterURL)
	if posterURL == "" {
		return fmt.Errorf("poster URL is required")
	}
	u := fmt.Sprintf("%s/library/metadata/%s/posters", c.baseURL, ratingKey)
	q := url.Values{}
	q.Set("url", posterURL)
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
		return fmt.Errorf("plex set poster url failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) UploadPosterData(ratingKey, dataURL string) error {
	if !c.Configured() {
		return fmt.Errorf("plex client not configured")
	}
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 || !strings.Contains(parts[0], ";base64") {
		return fmt.Errorf("invalid poster data")
	}
	blob, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("decode poster: %w", err)
	}
	mimeType := "image/jpeg"
	if strings.Contains(parts[0], "image/png") {
		mimeType = "image/png"
	}
	u := fmt.Sprintf("%s/library/metadata/%s/posters", c.baseURL, ratingKey)
	q := url.Values{}
	q.Set("X-Plex-Token", c.token)
	u += "?" + q.Encode()
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(blob))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mimeType)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("plex upload poster failed: %s", resp.Status)
	}
	return nil
}
