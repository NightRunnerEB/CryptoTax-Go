package worker

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OutboxDispatcher struct {
	cfg        config.RabbitConfig
	outboxRepo domain.OutboxRepo
	publisher  domain.EventPublisher
	log        *zap.Logger

	mu            sync.Mutex
	nextAttemptAt map[uuid.UUID]time.Time
}

func NewOutboxDispatcher(
	cfg config.RabbitConfig,
	outboxRepo domain.OutboxRepo,
	publisher domain.EventPublisher,
	log *zap.Logger,
) *OutboxDispatcher {
	return &OutboxDispatcher{
		cfg:           cfg,
		outboxRepo:    outboxRepo,
		publisher:     publisher,
		log:           log,
		nextAttemptAt: make(map[uuid.UUID]time.Time),
	}
}

func (d *OutboxDispatcher) Start(ctx context.Context) error {
	if d == nil {
		return nil
	}

	ticker := time.NewTicker(d.cfg.OutboxPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			d.dispatchPending(ctx)
		}
	}
}

func (d *OutboxDispatcher) dispatchPending(ctx context.Context) {
	events, err := d.outboxRepo.ListPending(ctx, d.cfg.OutboxMaxAttempts, d.cfg.OutboxBatchSize)
	if err != nil {
		d.log.Error("OutboxDispatcher: list pending failed", zap.Error(err))
		return
	}
	if len(events) == 0 {
		return
	}

	for _, event := range events {
		if !d.canAttempt(event.ID) {
			continue
		}

		routingKey := d.routingKeyFor(event.EventType)
		if strings.TrimSpace(routingKey) == "" {
			_ = d.outboxRepo.MarkPublishFailed(ctx, event.ID, d.cfg.OutboxMaxAttempts, "unknown event type")
			d.scheduleBackoff(event.ID, event.Attempts+1)
			continue
		}

		envelope := domain.BrokerEnvelope{
			EventType: event.EventType,
			Payload:   event.Payload,
		}
		body, err := json.Marshal(envelope)
		if err != nil {
			_ = d.outboxRepo.MarkPublishFailed(ctx, event.ID, d.cfg.OutboxMaxAttempts, err.Error())
			d.scheduleBackoff(event.ID, event.Attempts+1)
			continue
		}

		if err := d.publisher.Publish(ctx, routingKey, body); err != nil {
			_ = d.outboxRepo.MarkPublishFailed(ctx, event.ID, d.cfg.OutboxMaxAttempts, err.Error())
			d.scheduleBackoff(event.ID, event.Attempts+1)
			continue
		}

		if err := d.outboxRepo.MarkPublished(ctx, event.ID); err != nil {
			d.log.Error(
				"OutboxDispatcher: mark published failed",
				zap.Error(err),
				zap.String("event_id", event.ID.String()),
			)
		}
		d.clearBackoff(event.ID)
	}
}

func (d *OutboxDispatcher) routingKeyFor(eventType string) string {
	switch eventType {
	case domain.EventTypeReportRendered:
		return d.cfg.RoutingRendered
	case domain.EventTypeReportRenderFailed:
		return d.cfg.RoutingRenderFailed
	default:
		return ""
	}
}

func (d *OutboxDispatcher) canAttempt(eventID uuid.UUID) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	next, ok := d.nextAttemptAt[eventID]
	if !ok {
		return true
	}
	return !time.Now().Before(next)
}

func (d *OutboxDispatcher) scheduleBackoff(eventID uuid.UUID, attempts int32) {
	backoff := backoffForAttempts(attempts)
	d.mu.Lock()
	d.nextAttemptAt[eventID] = time.Now().Add(backoff)
	d.mu.Unlock()
}

func (d *OutboxDispatcher) clearBackoff(eventID uuid.UUID) {
	d.mu.Lock()
	delete(d.nextAttemptAt, eventID)
	d.mu.Unlock()
}

func backoffForAttempts(attempts int32) time.Duration {
	if attempts <= 1 {
		return time.Second
	}

	d := time.Second
	for i := int32(1); i < attempts && d < time.Minute; i++ {
		d *= 2
		if d > time.Minute {
			return time.Minute
		}
	}
	return d
}
