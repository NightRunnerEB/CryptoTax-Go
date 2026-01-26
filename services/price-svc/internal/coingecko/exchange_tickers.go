package coingecko

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type ExchangeTickersResponse struct {
	Name    string           `json:"name"`
	Tickers []ExchangeTicker `json:"tickers"`
}

type ExchangeTicker struct {
	Base   string `json:"base"`
	Target string `json:"target"`
}

type ExchangeTickersParams struct {
	CoinIDs             []string
	IncludeExchangeLogo *bool
	Page                *int
	// Depth               *bool
	// Order               *string
	// DexPairFormat       *string
}

func (c *CGClient) ExchangesTickers(
	ctx context.Context,
	exchangeID string,
	p ExchangeTickersParams,
) (*ExchangeTickersResponse, error) {
	q := url.Values{}
	if len(p.CoinIDs) > 0 {
		q.Set("coin_ids", strings.Join(p.CoinIDs, ","))
	}
	if p.IncludeExchangeLogo != nil {
		q.Set("include_exchange_logo", strconv.FormatBool(*p.IncludeExchangeLogo))
	}
	if p.Page != nil {
		q.Set("page", strconv.Itoa(*p.Page))
	}

	path := fmt.Sprintf("/exchanges/%s/tickers", url.PathEscape(exchangeID))

	var out ExchangeTickersResponse
	if err := c.doJSON(ctx, http.MethodGet, path, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
