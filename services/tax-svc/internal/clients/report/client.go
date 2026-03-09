package report

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	reportv1 "github.com/NightRunner/CryptoTax-Go/gen/report/v1"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type Client struct {
	addr    string
	timeout time.Duration
	conn    *grpc.ClientConn
	client  reportv1.ReportClient
}

func NewClient(ctx context.Context, addr string, timeout time.Duration) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, apperr.Internal("report grpc dial failed", err, map[string]string{
			"addr": addr,
		})
	}

	return &Client{
		addr:    addr,
		timeout: timeout,
		conn:    conn,
		client:  reportv1.NewReportClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) RequestRender(ctx context.Context, req domain.ReportRenderRequest) error {
	// Stub for MVP: report integration is not enabled yet in tax-svc.
	_ = ctx
	_ = req
	return nil
}

var _ domain.ReportClient = (*Client)(nil)
