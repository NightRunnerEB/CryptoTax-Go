package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

type CGClient struct {
	baseURL           *url.URL
	apiKey            string
	granularityPolicy GranularityPolicy

	httpClient *http.Client
	limiter    *rate.Limiter
}

func NewCGClient(cgConfig CGConfig) (*CGClient, error) {
	u, err := url.Parse(cgConfig.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid baseURL: %w", err)
	}

	limiter := rate.NewLimiter(rate.Limit(cgConfig.RateLimitPerMin)/60, cgConfig.RateLimitPerMin)
	return &CGClient{
		baseURL:           u,
		apiKey:            cgConfig.APIKey,
		granularityPolicy: cgConfig.GranularityPolicy,
		httpClient:        &http.Client{},
		limiter:           limiter,
	}, nil
}

func (c *CGClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-cg-demo-api-key", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http request returned status %d", resp.StatusCode)
	}

	return resp, nil
}

func (c *CGClient) doJSON(ctx context.Context, method, path string, q url.Values, out any) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait: %w", err)
	}

	u := c.baseURL.String() + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("x-cg-demo-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("coingecko http %d: %s", resp.StatusCode, string(body))
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("json unmarshal: %w; body=%s", err, string(body))
	}
	return nil
}

func (c *CGClient) GetGranularitySeconds(txTimeUTC, nowUTC time.Time) time.Duration {
	age := nowUTC.Sub(txTimeUTC)
	switch {
	case age < c.granularityPolicy["5minutes"]:
		return 300 * time.Second
	case age < c.granularityPolicy["1hour"]:
		return 3600 * time.Second
	default:
		return 86400 * time.Second
	}
}
