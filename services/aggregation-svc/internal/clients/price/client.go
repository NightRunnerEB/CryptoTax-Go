package price

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pricev1 "github.com/NightRunner/CryptoTax-Go/gen/price/v1"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/config"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

type Client struct {
	conn    *grpc.ClientConn
	client  pricev1.PriceClient
	timeout time.Duration
}

func NewClient(ctx context.Context, cfg config.Price) (*Client, error) {
	conn, err := grpc.NewClient(cfg.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, apperr.PriceUnavailable("price grpc dial failed", err, map[string]string{
			"addr": cfg.Addr,
		})
	}

	return &Client{
		conn:    conn,
		client:  pricev1.NewPriceClient(conn),
		timeout: cfg.Timeout,
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) ValuateTransactionsBatch(ctx context.Context, req *pricev1.ValuateTransactionsRequest) (*pricev1.ValuateTransactionsResponse, error) {
	if c == nil || c.client == nil {
		return nil, apperr.Internal("price client is not initialized", nil, nil)
	}

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	return c.client.ValuateTransactionsBatch(ctx, req)
}
