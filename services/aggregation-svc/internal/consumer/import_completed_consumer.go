package consumer

import (
	"context"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"go.uber.org/zap"
)

type ImportCompletedConsumer struct {
	cfg config.RabbitMQ
	uc  domain.AggregationUseCase
	log *zap.Logger
}

func NewImportCompletedConsumer(cfg config.RabbitMQ, uc domain.AggregationUseCase, log *zap.Logger) *ImportCompletedConsumer {
	return &ImportCompletedConsumer{
		cfg: cfg,
		uc:  uc,
		log: log,
	}
}

func (c *ImportCompletedConsumer) Start(ctx context.Context) error {
	if c == nil {
		return apperr.Internal("consumer is nil", nil, nil)
	}

	c.log.Info("ImportCompletedConsumer: not implemented; waiting for ctx")
	<-ctx.Done()
	return nil
}

func (c *ImportCompletedConsumer) Close() error {
	return nil
}
