package winston

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type QueueListResponse struct {
	Success bool                `json:"success"`
	Data    []QueueItemResponse `json:"data"`
}

type QueueItemResponse struct {
	ID         int    `json:"id"`
	Status     string `json:"status"`
	NZBPath    string `json:"nzb_path"`
	TargetPath string `json:"target_path"`
	Percentage int    `json:"percentage"`
}

func (c *AltMountClient) GetQueueItem(ctx context.Context, queueID int) (*QueueItemResponse, error) {
	u := c.baseURL + "/api/queue"
	q := url.Values{}
	q.Set("limit", "200")
	if c.apiKey != "" {
		q.Set("apikey", c.apiKey)
	}
	u += "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("altmount queue lookup failed: %s", resp.Status)
	}

	var out QueueListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	for _, item := range out.Data {
		if item.ID == queueID {
			cp := item
			return &cp, nil
		}
	}
	return nil, nil
}

func (p *ImportProcessor) waitForQueueReady(ctx context.Context, queueID int) (*QueueItemResponse, error) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	deadline := time.NewTimer(30 * time.Minute)
	defer deadline.Stop()

	for {
		item, err := p.alt.GetQueueItem(ctx, queueID)
		if err == nil && item != nil {
			switch item.Status {
			case "completed":
				return item, nil
			case "failed":
				return item, fmt.Errorf("altmount queue item %d failed", queueID)
			}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, fmt.Errorf("timeout waiting queue item %d", queueID)
		case <-ticker.C:
		}
	}
}
