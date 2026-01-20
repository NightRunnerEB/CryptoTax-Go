package fiatfx

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

func FetchXML[T any](
	ctx context.Context,
	httpClient *http.Client,
	url string,
) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("bad status: %s (%s)", resp.Status, strings.TrimSpace(string(b)))
	}

	var out T
	dec := xml.NewDecoder(resp.Body)
	dec.CharsetReader = charset.NewReaderLabel
	if err := dec.Decode(&out); err != nil {
		return nil, err
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
