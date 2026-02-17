package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"github.com/google/uuid"
	rabbitmq "github.com/wagslane/go-rabbitmq"
	"go.uber.org/zap"
)

type ImportCompletedConsumer struct {
	cfg       config.RabbitMQ
	uc        domain.AggregationUseCase
	log       *zap.Logger
	conn      *rabbitmq.Conn
	consumer  *rabbitmq.Consumer
	closeOnce sync.Once
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
	if c.uc == nil {
		return apperr.Internal("aggregation usecase is nil", nil, nil)
	}

	conn, err := rabbitmq.NewConn(
		c.cfg.URL,
		rabbitmq.WithConnectionOptionsReconnectInterval(c.cfg.ReconnectInterval),
		rabbitmq.WithConnectionOptionsLogging,
	)
	if err != nil {
		return apperr.Internal("rabbitmq connect failed", err, map[string]string{
			"url": c.cfg.URL,
		})
	}
	c.conn = conn

	consumerOptions := []func(*rabbitmq.ConsumerOptions){
		rabbitmq.WithConsumerOptionsLogging,
		rabbitmq.WithConsumerOptionsQOSPrefetch(c.cfg.Prefetch),
		rabbitmq.WithConsumerOptionsConcurrency(c.cfg.Concurrency),
		rabbitmq.WithConsumerOptionsExchangeName(c.cfg.Exchange),
		rabbitmq.WithConsumerOptionsRoutingKey(c.cfg.RoutingKey),
		rabbitmq.WithConsumerOptionsConsumerName(c.cfg.ConsumerName),
	}
	if c.cfg.QueueDurable {
		consumerOptions = append(consumerOptions, rabbitmq.WithConsumerOptionsQueueDurable)
	}

	rmqConsumer, err := rabbitmq.NewConsumer(c.conn, c.cfg.Queue, consumerOptions...)
	if err != nil {
		return apperr.Internal("rabbitmq consumer init failed", err, map[string]string{
			"queue": c.cfg.Queue,
		})
	}
	c.consumer = rmqConsumer

	go func() {
		<-ctx.Done()
		_ = c.Close()
	}()

	c.log.Info("ImportCompletedConsumer: started",
		zap.String("exchange", c.cfg.Exchange),
		zap.String("queue", c.cfg.Queue),
		zap.String("routing_key", c.cfg.RoutingKey),
		zap.Int("prefetch", c.cfg.Prefetch),
		zap.Int("concurrency", c.cfg.Concurrency),
	)

	if err := c.consumer.Run(c.handleDelivery); err != nil {
		return apperr.Internal("rabbitmq consumer stopped with error", err, map[string]string{
			"queue": c.cfg.Queue,
		})
	}

	return nil
}

func (c *ImportCompletedConsumer) Close() error {
	if c == nil {
		return nil
	}

	var closeErr error
	c.closeOnce.Do(func() {
		if c.consumer != nil {
			c.consumer.Close()
		}
		if c.conn != nil {
			closeErr = c.conn.Close()
		}
	})
	return closeErr
}

func (c *ImportCompletedConsumer) handleDelivery(d rabbitmq.Delivery) rabbitmq.Action {
	event, err := decodeImportCompletedEvent(d.Body)
	if err != nil {
		c.log.Warn("ImportCompletedConsumer: invalid message payload",
			zap.Error(err),
			zap.ByteString("body", d.Body),
		)
		return rabbitmq.NackDiscard
	}

	if err := c.uc.ProcessImport(context.Background(), event); err != nil {
		if shouldRequeue(err) {
			c.log.Warn("ImportCompletedConsumer: process import failed, requeue",
				zap.Error(err),
				zap.String("tenant_id", event.TenantID.String()),
				zap.String("import_id", event.ImportID.String()),
			)
			return rabbitmq.NackRequeue
		}
		c.log.Warn("ImportCompletedConsumer: process import failed, discard",
			zap.Error(err),
			zap.String("tenant_id", event.TenantID.String()),
			zap.String("import_id", event.ImportID.String()),
		)
		return rabbitmq.NackDiscard
	}

	return rabbitmq.Ack
}

type importCompletedEnvelope struct {
	ImportCompleted *domain.ImportEvent `json:"ImportCompleted"`
}

func decodeImportCompletedEvent(body []byte) (domain.ImportEvent, error) {
	var envelope importCompletedEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.ImportCompleted != nil {
		event := *envelope.ImportCompleted
		if event.TenantID == uuid.Nil || event.ImportID == uuid.Nil {
			return domain.ImportEvent{}, fmt.Errorf("missing tenant_id or import_id in ImportCompleted payload")
		}
		return event, nil
	}

	var event domain.ImportEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return domain.ImportEvent{}, fmt.Errorf("decode import event: %w", err)
	}
	if event.TenantID == uuid.Nil || event.ImportID == uuid.Nil {
		return domain.ImportEvent{}, fmt.Errorf("missing tenant_id or import_id")
	}

	return event, nil
}

func shouldRequeue(err error) bool {
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		return true
	}

	switch ae.Code {
	case apperr.ErrImportAlreadyDone:
		return false
	case apperr.ErrInvalidArgument:
		return false
	case apperr.ErrLedgerUnavailable, apperr.ErrLedgerBadResponse, apperr.ErrPriceUnavailable, apperr.ErrPriceBadResponse:
		return true
	case apperr.ErrImportLocked, apperr.ErrImportInconsistent, apperr.ErrInternal:
		return true
	default:
		return true
	}
}
