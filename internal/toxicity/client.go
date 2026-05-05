package toxicity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"dxcluster/spot"
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	endpoint string
	token    string
	http     httpDoer
}

func NewClient(cfg Config, httpClient httpDoer) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		endpoint: strings.TrimSpace(cfg.Endpoint),
		token:    strings.TrimSpace(os.Getenv(strings.TrimSpace(cfg.BearerTokenEnv))),
		http:     httpClient,
	}
}

func (c *Client) Classify(ctx context.Context, comment string) (Decision, error) {
	if c == nil || c.endpoint == "" || c.token == "" {
		return Decision{}, fmt.Errorf("toxicity client is not configured")
	}
	payload, err := json.Marshal(struct {
		Comment string `json:"comment"`
	}{Comment: comment})
	if err != nil {
		return Decision{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return Decision{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return Decision{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return Decision{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Decision{}, fmt.Errorf("toxicity worker HTTP %s", resp.Status)
	}
	var out struct {
		Status     string   `json:"status"`
		Categories []string `json:"categories"`
		Model      string   `json:"model"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return Decision{}, err
	}
	switch strings.ToLower(strings.TrimSpace(out.Status)) {
	case "safe":
		return Decision{Status: spot.ToxicitySafe, Categories: cleanCategories(out.Categories), Model: strings.TrimSpace(out.Model)}, nil
	case "toxic":
		return Decision{Status: spot.ToxicityToxic, Categories: cleanCategories(out.Categories), Model: strings.TrimSpace(out.Model)}, nil
	default:
		return Decision{}, fmt.Errorf("toxicity worker returned unknown status %q", out.Status)
	}
}

func cleanCategories(categories []string) []string {
	out := make([]string, 0, len(categories))
	for _, category := range categories {
		category = strings.TrimSpace(category)
		if category != "" {
			out = append(out, category)
		}
	}
	return out
}
