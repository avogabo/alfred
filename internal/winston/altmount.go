package winston

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"alfred/internal/config"
)

type AltMountClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

type ManualImportRequest struct {
	FilePath     string  `json:"file_path"`
	RelativePath *string `json:"relative_path,omitempty"`
}

type APIEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

type ManualImportResponse struct {
	Message string `json:"message"`
	QueueID int    `json:"queue_id"`
}

func NewAltMountClient(cfg config.Config) *AltMountClient {
	return &AltMountClient{
		baseURL: strings.TrimRight(cfg.AltMountBaseURL, "/"),
		apiKey:  cfg.AltMountAPIKey,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *AltMountClient) ImportFile(ctx context.Context, req ManualImportRequest) (*ManualImportResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	fmt.Printf("winston altmount payload: %s\n", string(payload))

	u := c.baseURL + "/api/import/file"
	if c.apiKey != "" {
		q := url.Values{}
		q.Set("apikey", c.apiKey)
		u += "?" + q.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("altmount import failed: %s", msg)
	}

	var env APIEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}

	var out ManualImportResponse
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &out); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func (c *AltMountClient) BuildImportRequest(sourceNZB string, mappedFilePath string, relativePath string) ManualImportRequest {
	derivedRelative := ""
	if strings.TrimSpace(sourceNZB) != "" && strings.TrimSpace(mappedFilePath) != "" {
		base := filepath.Dir(filepath.Clean(mappedFilePath))
		if rel, err := filepath.Rel(base, filepath.Clean(mappedFilePath)); err == nil && !strings.HasPrefix(rel, "..") {
			_ = rel
		}
	}
	if strings.TrimSpace(relativePath) != "" {
		derivedRelative = strings.TrimSpace(relativePath)
	}
	var relPtr *string
	if derivedRelative != "" {
		rel := derivedRelative
		relPtr = &rel
	}
	return ManualImportRequest{FilePath: mappedFilePath, RelativePath: relPtr}
}
