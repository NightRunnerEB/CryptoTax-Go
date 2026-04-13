package ledger

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

type Client struct {
	baseURL string
	http    *resty.Client
}

func NewClient(cfg config.Ledger) *Client {
	http := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetTimeout(cfg.Timeout)

	return &Client{
		baseURL: cfg.BaseURL,
		http:    http,
	}
}

func (c *Client) ListTransactionsByImport(ctx context.Context, userID, importID uuid.UUID) ([]domain.LedgerTransaction, error) {
	if c == nil || c.http == nil {
		return nil, apperr.Internal("ledger client is not initialized", nil, nil)
	}

	path := fmt.Sprintf("/v1/users/%s/imports/%s/transactions", userID, importID)
	var out []domain.LedgerTransaction

	resp, err := c.http.R().
		SetContext(ctx).
		SetResult(&out).
		Get(path)

	if err != nil {
		return nil, apperr.LedgerUnavailable("ledger request failed", err, map[string]string{
			"path": path,
		})
	}
	if resp.IsError() {
		return nil, apperr.LedgerBadResponse("ledger bad response", nil, map[string]string{
			"status":      resp.Status(),
			"status_code": strconv.Itoa(resp.StatusCode()),
			"path":        path,
		})
	}

	return out, nil
}
