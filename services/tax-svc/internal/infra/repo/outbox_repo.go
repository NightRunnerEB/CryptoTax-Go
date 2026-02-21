package repository

import (
	"context"
	"time"

	db "github.com/NightRunner/CryptoTax-Go/services/tax-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/google/uuid"
)

type outboxRepo struct {
	store db.Store
}

func NewOutboxRepo(store db.Store) domain.OutboxRepo {
	return &outboxRepo{store: store}
}

func (r *outboxRepo) Insert(ctx context.Context, event domain.OutboxEvent) error {
	if err := r.store.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
		ID:            event.ID,
		AggregateType: event.AggregateType,
		AggregateID:   event.AggregateID,
		EventType:     event.EventType,
		Payload:       event.Payload,
		Status:        string(event.Status),
	}); err != nil {
		return apperr.Internal("insert outbox event failed", err, map[string]string{
			"event_id": event.ID.String(),
		})
	}
	return nil
}

func (r *outboxRepo) ListPending(ctx context.Context, maxAttempts, limit int32) ([]domain.OutboxEvent, error) {
	rows, err := r.store.ListPendingOutboxEvents(ctx, db.ListPendingOutboxEventsParams{
		Attempts: maxAttempts,
		Limit:    limit,
	})
	if err != nil {
		return nil, apperr.Internal("list outbox events failed", err, nil)
	}

	out := make([]domain.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		var publishedAt *time.Time
		if row.PublishedAt.Valid {
			v := fromTimestamptz(row.PublishedAt)
			publishedAt = &v
		}
		out = append(out, domain.OutboxEvent{
			ID:            row.ID,
			AggregateType: row.AggregateType,
			AggregateID:   row.AggregateID,
			EventType:     row.EventType,
			Payload:       row.Payload,
			CreatedAt:     fromTimestamptz(row.CreatedAt),
			PublishedAt:   publishedAt,
			Status:        domain.OutboxStatus(row.Status),
			Attempts:      row.Attempts,
			LastError:     row.LastError,
		})
	}
	return out, nil
}

func (r *outboxRepo) MarkPublished(ctx context.Context, id uuid.UUID) error {
	if err := r.store.MarkOutboxEventPublished(ctx, id); err != nil {
		return apperr.Internal("mark outbox published failed", err, map[string]string{
			"event_id": id.String(),
		})
	}
	return nil
}

func (r *outboxRepo) MarkPublishFailed(ctx context.Context, id uuid.UUID, maxAttempts int32, lastError string) error {
	if err := r.store.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
		ID:        id,
		Attempts:  maxAttempts,
		LastError: &lastError,
	}); err != nil {
		return apperr.Internal("mark outbox failed failed", err, map[string]string{
			"event_id": id.String(),
		})
	}
	return nil
}

var _ domain.OutboxRepo = (*outboxRepo)(nil)
