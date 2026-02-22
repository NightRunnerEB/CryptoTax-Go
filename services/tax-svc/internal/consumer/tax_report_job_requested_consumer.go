package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/google/uuid"
	rabbitmq "github.com/wagslane/go-rabbitmq"
	"go.uber.org/zap"
)

type TaxReportJobRequestedConsumer struct {
	cfg      config.RabbitConfig
	uc       domain.ReportPipelineUseCase
	log      *zap.Logger
	conn     *rabbitmq.Conn
	consumer *rabbitmq.Consumer
	once     sync.Once
}

func NewTaxReportJobRequestedConsumer(cfg config.RabbitConfig, uc domain.ReportPipelineUseCase, log *zap.Logger) *TaxReportJobRequestedConsumer {
	return &TaxReportJobRequestedConsumer{
		cfg: cfg,
		uc:  uc,
		log: log,
	}
}

func (c *TaxReportJobRequestedConsumer) Start(ctx context.Context) error {
	if c == nil {
		return apperr.Internal("consumer is nil", nil, nil)
	}
	if c.uc == nil {
		return apperr.Internal("report pipeline usecase is nil", nil, nil)
	}

	conn, err := rabbitmq.NewConn(
		c.cfg.URL,
		rabbitmq.WithConnectionOptionsReconnectInterval(c.cfg.ReconnectInterval),
		rabbitmq.WithConnectionOptionsLogging,
	)
	if err != nil {
		return apperr.Internal("rabbitmq connect failed", err, nil)
	}
	c.conn = conn

	options := []func(*rabbitmq.ConsumerOptions){
		rabbitmq.WithConsumerOptionsLogging,
		rabbitmq.WithConsumerOptionsQOSPrefetch(c.cfg.Prefetch),
		rabbitmq.WithConsumerOptionsConcurrency(c.cfg.Concurrency),
		rabbitmq.WithConsumerOptionsConsumerName(c.cfg.ConsumerNameJobs),
	}
	if c.cfg.QueueDurable {
		options = append(options, rabbitmq.WithConsumerOptionsQueueDurable)
	}
	if c.cfg.SkipQueueDeclare {
		options = append(options, rabbitmq.WithConsumerOptionsQueueNoDeclare)
	}

	consumer, err := rabbitmq.NewConsumer(conn, c.cfg.QueueJobRequested, options...)
	if err != nil {
		return apperr.Internal("rabbitmq consumer init failed", err, map[string]string{
			"queue": c.cfg.QueueJobRequested,
		})
	}
	c.consumer = consumer

	go func() {
		<-ctx.Done()
		_ = c.Close()
	}()

	if err := c.consumer.Run(c.handleDelivery); err != nil {
		return apperr.Internal("rabbitmq consumer run failed", err, map[string]string{
			"queue": c.cfg.QueueJobRequested,
		})
	}
	return nil
}

func (c *TaxReportJobRequestedConsumer) Close() error {
	if c == nil {
		return nil
	}

	var closeErr error
	c.once.Do(func() {
		if c.consumer != nil {
			c.consumer.Close()
		}
		if c.conn != nil {
			closeErr = c.conn.Close()
		}
	})
	return closeErr
}

func (c *TaxReportJobRequestedConsumer) handleDelivery(d rabbitmq.Delivery) rabbitmq.Action {
	event, err := decodeTaxReportJobRequested(d)
	if err != nil {
		c.log.Warn("TaxReportJobRequestedConsumer: invalid message", zap.Error(err), zap.ByteString("body", d.Body))
		return rabbitmq.NackDiscard
	}

	callCtx := context.Background()
	if c.cfg.HandlerTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(callCtx, c.cfg.HandlerTimeout)
		defer cancel()
	}

	if err := c.uc.ProcessQueuedReport(callCtx, event); err != nil {
		if shouldRequeue(err) {
			c.log.Warn("TaxReportJobRequestedConsumer: process failed, requeue", zap.Error(err))
			return rabbitmq.NackRequeue
		}
		c.log.Warn("TaxReportJobRequestedConsumer: process failed, discard", zap.Error(err))
		return rabbitmq.NackDiscard
	}

	return rabbitmq.Ack
}

func decodeTaxReportJobRequested(d rabbitmq.Delivery) (domain.TaxReportJobRequestedEvent, error) {
	parsedEventType := ""
	payload := d.Body

	var envelope domain.BrokerEnvelope
	if err := json.Unmarshal(d.Body, &envelope); err == nil && len(envelope.Payload) > 0 {
		parsedEventType = envelope.EventType
		payload = envelope.Payload
	}

	if parsedEventType == "" {
		parsedEventType = d.RoutingKey
	}
	if parsedEventType != "" &&
		parsedEventType != domain.EventTypeTaxReportJobRequested &&
		parsedEventType != "TaxReportJobRequested" {
		return domain.TaxReportJobRequestedEvent{}, fmt.Errorf("unexpected event type: %s", parsedEventType)
	}

	var event domain.TaxReportJobRequestedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return domain.TaxReportJobRequestedEvent{}, fmt.Errorf("decode payload: %w", err)
	}
	if event.EventID == uuid.Nil || event.ReportID == uuid.Nil || event.TenantID == uuid.Nil {
		return domain.TaxReportJobRequestedEvent{}, fmt.Errorf("missing ids in payload")
	}
	return event, nil
}

func shouldRequeue(err error) bool {
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		return true
	}

	switch ae.Code {
	case apperr.ErrInvalidArgument,
		apperr.ErrNeedsPriceResolution,
		apperr.ErrNotFound,
		apperr.ErrNegativeInventory,
		apperr.ErrInvalidTxShape,
		apperr.ErrUnsupportedKind:
		return false
	case apperr.ErrAggregationUnavailable,
		apperr.ErrAggregationFetchFailed,
		apperr.ErrStorageUnavailable,
		apperr.ErrAggregationBadResponse,
		apperr.ErrStorageBadResponse,
		apperr.ErrMinIOUploadFailed,
		apperr.ErrRabbitPublishFailed:
		return true
	default:
		return true
	}
}
