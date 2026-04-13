package repository

import (
	"context"

	"github.com/google/uuid"

	db "github.com/NightRunner/CryptoTax-Go/services/report-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
)

type inboxRepo struct {
	store db.Store
}

func NewInboxRepo(store db.Store) domain.InboxRepo {
	return &inboxRepo{store: store}
}

func (r *inboxRepo) Register(ctx context.Context, eventID uuid.UUID) (bool, error) {
	rows, err := r.store.InsertInboxEvent(ctx, eventID)
	if err != nil {
		return false, apperr.Internal("register inbox event failed", err, map[string]string{
			"event_id": eventID.String(),
		})
	}
	return rows > 0, nil
}

var _ domain.InboxRepo = (*inboxRepo)(nil)
