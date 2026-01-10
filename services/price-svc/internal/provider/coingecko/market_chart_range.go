package coingecko

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type MarketChartRangeResponse struct {
	Prices [][]float64 `json:"prices"` // [ [ts_ms, price], ... ]
}

func (c *CGClient) CoinsMarketChartRange(
	ctx context.Context,
	coinID string,
	vsCurrency string,
	from time.Time,
	to time.Time,
	precision *string,
) (*MarketChartRangeResponse, error) {
	q := url.Values{}
	q.Set("vs_currency", vsCurrency)
	q.Set("from", strconv.FormatInt(from.UTC().Unix(), 10))
	q.Set("to", strconv.FormatInt(to.UTC().Unix(), 10))
	if precision != nil {
		q.Set("precision", *precision)
	}

	path := fmt.Sprintf("/coins/%s/market_chart/range", url.PathEscape(coinID))

	var out MarketChartRangeResponse
	if err := c.doJSON(ctx, http.MethodGet, path, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
