package rabbit

import (
	"context"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	rabbitmq "github.com/wagslane/go-rabbitmq"
)

type Publisher struct {
	cfg  config.RabbitConfig
	conn *rabbitmq.Conn
	pub  *rabbitmq.Publisher
}

func NewPublisher(cfg config.RabbitConfig) (*Publisher, error) {
	conn, err := rabbitmq.NewConn(
		cfg.URL,
		rabbitmq.WithConnectionOptionsReconnectInterval(cfg.ReconnectInterval),
		rabbitmq.WithConnectionOptionsLogging,
	)
	if err != nil {
		return nil, apperr.Internal("rabbitmq connect failed", err, nil)
	}

	pub, err := rabbitmq.NewPublisher(
		conn,
		rabbitmq.WithPublisherOptionsLogging,
		rabbitmq.WithPublisherOptionsExchangeName(cfg.Exchange),
	)
	if err != nil {
		_ = conn.Close()
		return nil, apperr.Internal("rabbitmq publisher init failed", err, nil)
	}

	return &Publisher{
		cfg:  cfg,
		conn: conn,
		pub:  pub,
	}, nil
}

func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	if p == nil || p.pub == nil {
		return apperr.Internal("rabbitmq publisher is not initialized", nil, nil)
	}
	return p.pub.PublishWithContext(
		ctx,
		body,
		[]string{routingKey},
		rabbitmq.WithPublishOptionsExchange(p.cfg.Exchange),
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsPersistentDelivery,
	)
}

func (p *Publisher) Close() error {
	if p == nil {
		return nil
	}
	if p.pub != nil {
		p.pub.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

var _ domain.EventPublisher = (*Publisher)(nil)
