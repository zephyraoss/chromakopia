package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) RecordingSummary(ctx context.Context, mbid string) (*RecordingSummary, error) {
	u := c.baseURL + "/catalog/recording/" + url.PathEscape(mbid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("metadata request %s: %w", mbid, err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("metadata request %s: %w", mbid, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, nil
	default:
		return nil, fmt.Errorf("metadata request %s: unexpected status %d", mbid, resp.StatusCode)
	}

	var rec Recording
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&rec); err != nil {
		return nil, fmt.Errorf("metadata response %s: %w", mbid, err)
	}
	return summarizeRecording(&rec), nil
}
