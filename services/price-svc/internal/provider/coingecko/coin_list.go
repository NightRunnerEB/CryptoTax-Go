package coingecko

import (
	"context"
	"net/http"
	"net/url"
)

type CoinListItem struct {
	ID        string            `json:"id"`
	Symbol    string            `json:"symbol"`
	Name      string            `json:"name"`
}

func (c *CGClient) CoinsList(ctx context.Context, includePlatform bool) ([]CoinListItem, error) {
	q := url.Values{}
	if includePlatform {
		q.Set("include_platform", "true")
	}

	var out []CoinListItem
	if err := c.doJSON(ctx, http.MethodGet, "/coins/list", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}
