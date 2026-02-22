package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/google/uuid"
	rabbitmq "github.com/wagslane/go-rabbitmq"
	"go.uber.org/zap"
)

type ReportRenderEventsConsumer struct {
	cfg      config.RabbitConfig
	uc       domain.ReportPipelineUseCase
	log      *zap.Logger
	conn     *rabbitmq.Conn
	consumer *rabbitmq.Consumer
	once     sync.Once
}

func NewReportRenderEventsConsumer(cfg config.RabbitConfig, uc domain.ReportPipelineUseCase, log *zap.Logger) *ReportRenderEventsConsumer {
	return &ReportRenderEventsConsumer{
		cfg: cfg,
		uc:  uc,
		log: log,
	}
}

func (c *ReportRenderEventsConsumer) Start(ctx context.Context) error {
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
		rabbitmq.WithConsumerOptionsConsumerName(c.cfg.ConsumerNameResults),
	}
	if c.cfg.QueueDurable {
		options = append(options, rabbitmq.WithConsumerOptionsQueueDurable)
	}
	if c.cfg.SkipQueueDeclare {
		options = append(options, rabbitmq.WithConsumerOptionsQueueNoDeclare)
	}

	consumer, err := rabbitmq.NewConsumer(conn, c.cfg.QueueRenderResults, options...)
	if err != nil {
		return apperr.Internal("rabbitmq consumer init failed", err, map[string]string{
			"queue": c.cfg.QueueRenderResults,
		})
	}
	c.consumer = consumer

	go func() {
		<-ctx.Done()
		_ = c.Close()
	}()

	if err := c.consumer.Run(c.handleDelivery); err != nil {
		return apperr.Internal("rabbitmq consumer run failed", err, map[string]string{
			"queue": c.cfg.QueueRenderResults,
		})
	}
	return nil
}

func (c *ReportRenderEventsConsumer) Close() error {
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

func (c *ReportRenderEventsConsumer) handleDelivery(d rabbitmq.Delivery) rabbitmq.Action {
	eventType, payload, err := decodeEnvelope(d)
	if err != nil {
		c.log.Warn("ReportRenderEventsConsumer: invalid message", zap.Error(err), zap.ByteString("body", d.Body))
		return rabbitmq.NackDiscard
	}

	callCtx := context.Background()
	if c.cfg.HandlerTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(callCtx, c.cfg.HandlerTimeout)
		defer cancel()
	}

	switch eventType {
	case domain.EventTypeReportRendered, c.cfg.RoutingRendered:
		var event domain.ReportRenderedEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			c.log.Warn("ReportRenderEventsConsumer: invalid rendered payload", zap.Error(err))
			return rabbitmq.NackDiscard
		}
		if event.EventID == uuid.Nil || event.ReportID == uuid.Nil {
			c.log.Warn("ReportRenderEventsConsumer: missing ids in rendered payload")
			return rabbitmq.NackDiscard
		}
		if err := c.uc.HandleReportRendered(callCtx, event); err != nil {
			c.log.Warn("ReportRenderEventsConsumer: handle rendered failed", zap.Error(err))
			return rabbitmq.NackRequeue
		}
		return rabbitmq.Ack

	case domain.EventTypeReportRenderFailed, c.cfg.RoutingRenderFailed:
		var event domain.ReportRenderFailedEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			c.log.Warn("ReportRenderEventsConsumer: invalid failed payload", zap.Error(err))
			return rabbitmq.NackDiscard
		}
		if event.EventID == uuid.Nil || event.ReportID == uuid.Nil {
			c.log.Warn("ReportRenderEventsConsumer: missing ids in failed payload")
			return rabbitmq.NackDiscard
		}
		if err := c.uc.HandleReportRenderFailed(callCtx, event); err != nil {
			c.log.Warn("ReportRenderEventsConsumer: handle failed event failed", zap.Error(err))
			return rabbitmq.NackRequeue
		}
		return rabbitmq.Ack
	default:
		c.log.Warn("ReportRenderEventsConsumer: unsupported event type", zap.String("event_type", eventType))
		return rabbitmq.NackDiscard
	}
}

func decodeEnvelope(d rabbitmq.Delivery) (string, []byte, error) {
	eventType := d.RoutingKey
	payload := d.Body

	var envelope domain.BrokerEnvelope
	if err := json.Unmarshal(d.Body, &envelope); err == nil && len(envelope.Payload) > 0 {
		payload = envelope.Payload
		if envelope.EventType != "" {
			eventType = envelope.EventType
		}
	}

	if eventType == "" {
		return "", nil, fmt.Errorf("missing event type")
	}
	return eventType, payload, nil
}
