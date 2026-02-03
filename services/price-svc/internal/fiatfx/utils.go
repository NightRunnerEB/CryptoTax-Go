package fiatfx

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"time"

	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
	"golang.org/x/net/html/charset"
)

func FetchXML[T any](
	ctx context.Context,
	httpClient *http.Client,
	url string,
) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, apperr.Internal("create http request failed", err, map[string]string{
			"url": url,
		})
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, apperr.Internal("http request failed", err, map[string]string{
			"url": url,
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, apperr.Internal("unexpected http status", nil, map[string]string{
			"url":    url,
			"status": resp.Status,
			"body":   strings.TrimSpace(string(b)),
		})
	}

	var out T
	dec := xml.NewDecoder(resp.Body)
	dec.CharsetReader = charset.NewReaderLabel
	if err := dec.Decode(&out); err != nil {
		return nil, apperr.Internal("decode xml failed", err, map[string]string{
			"url": url,
		})
	}

	return &out, nil
}

func dateOnly(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func dateKeyISO(t time.Time) string {
	return t.Format("2006-01-02")
}

func nextRunTime(now time.Time, s Schedule) time.Time {
	n := now.In(s.Loc)

	run := time.Date(n.Year(), n.Month(), n.Day(), s.Hour, s.Min, 0, 0, s.Loc)
	if !run.After(n) {
		run = run.AddDate(0, 0, 1)
	}
	return run
}
