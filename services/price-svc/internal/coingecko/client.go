package coingecko

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	applogger "github.com/NightRunner/CryptoTax-Go/pkg/logger"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

type CGClient struct {
	baseURL           *url.URL
	apiKey            string
	granularityPolicy GranularityPolicy

	httpClient *http.Client
	limiter    *rate.Limiter
}

const providerCoinGecko = "coingecko"

func NewCGClient(cgConfig CGConfig) (*CGClient, error) {
	u, err := url.Parse(cgConfig.BaseURL)
	if err != nil {
		return nil, apperr.InvalidArgument("invalid base url", err, apperr.FieldViolation{
			Field:       "coingecko.base_url",
			Description: "invalid format",
		})
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

func (c *CGClient) doJSON(ctx context.Context, method, path string, q url.Values, out any) error {
	log := applogger.FromContext(ctx)
	start := time.Now()

	if err := c.limiter.Wait(ctx); err != nil {
		log.Debug("coingecko rate limit wait failed", zap.Error(err))
		return apperr.ProviderUnavailable("rate limit wait failed", providerCoinGecko, err, nil)
	}

	u := c.baseURL.String() + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return apperr.Internal("create http request failed", err, map[string]string{
			"url": u,
		})
	}

	req.Header.Set("x-cg-demo-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	log.Debug(
		"coingecko request",
		zap.String("method", method),
		zap.String("url", u),
	)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return apperr.ProviderUnavailable("http request failed", providerCoinGecko, err, map[string]string{
			"url": u,
		})
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Debug("coingecko read response body failed", zap.Error(err))
		return apperr.ProviderUnavailable("read response body failed", providerCoinGecko, err, map[string]string{
			"url": u,
		})
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Debug(
			"coingecko http request non-2xx",
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("duration", time.Since(start)),
		)
		return apperr.ProviderBadResponse("unexpected http status", providerCoinGecko, nil, map[string]string{
			"status_code": strconv.Itoa(resp.StatusCode),
			"body":        truncate(string(body), 512),
		})
	}

	if out == nil {
		log.Debug(
			"coingecko request ok (no body decode)",
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("duration", time.Since(start)),
		)
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return apperr.ProviderBadResponse("json unmarshal failed", providerCoinGecko, err, map[string]string{
			"body": truncate(string(body), 512),
		})
	}

	log.Debug(
		"coingecko request ok",
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", time.Since(start)),
	)
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
