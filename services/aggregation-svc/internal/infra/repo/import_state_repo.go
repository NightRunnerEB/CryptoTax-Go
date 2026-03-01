package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

type importStateRepo struct {
	store db.Store
}

func NewImportStateRepo(store db.Store) domain.ImportStateRepo {
	return &importStateRepo{store: store}
}

func (r *importStateRepo) Get(ctx context.Context, tenantID, importID uuid.UUID) (domain.AggregationImportState, error) {
	row, err := r.store.GetAggregationImportState(ctx, db.GetAggregationImportStateParams{
		TenantID: tenantID,
		ImportID: importID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AggregationImportState{}, apperr.NotFound("import state not found", apperr.Resource{
				Type: "aggregation_import_state",
				Name: tenantID.String() + ":" + importID.String(),
			}, err)
		}
		return domain.AggregationImportState{}, apperr.Internal("get import state failed", err, map[string]string{
			"tenant_id": tenantID.String(),
			"import_id": importID.String(),
		})
	}

	state := domain.AggregationImportState{
		TenantID:    row.TenantID,
		ImportID:    row.ImportID,
		EventId:     row.EventID,
		Status:      domain.ImportStatus(row.Status),
		StartedAt:   fromTimestamptz(row.StartedAt),
		CompletedAt: nil,
		Error:       row.Error,
	}
	if row.CompletedAt.Valid {
		v := fromTimestamptz(row.CompletedAt)
		state.CompletedAt = &v
	}

	return state, nil
}

func (r *importStateRepo) UpsertProcessing(ctx context.Context, state domain.AggregationImportState) error {
	if err := r.store.UpsertAggregationImportStateProcessing(ctx, db.UpsertAggregationImportStateProcessingParams{
		TenantID: state.TenantID,
		ImportID: state.ImportID,
		EventID:  state.EventId,
		Status:   string(state.Status),
	}); err != nil {
		return apperr.Internal("upsert import processing failed", err, map[string]string{
			"tenant_id": state.TenantID.String(),
			"import_id": state.ImportID.String(),
		})
	}
	return nil
}

func (r *importStateRepo) MarkCompleted(ctx context.Context, tenantID, importID uuid.UUID) error {
	if err := r.store.MarkAggregationImportStateCompleted(ctx, db.MarkAggregationImportStateCompletedParams{
		TenantID: tenantID,
		ImportID: importID,
	}); err != nil {
		return apperr.Internal("mark import completed failed", err, map[string]string{
			"tenant_id": tenantID.String(),
			"import_id": importID.String(),
		})
	}
	return nil
}

func (r *importStateRepo) MarkFailed(ctx context.Context, tenantID, importID uuid.UUID, errMsg string) error {
	msg := errMsg
	if msg == "" {
		msg = "unknown error"
	}
	if err := r.store.MarkAggregationImportStateFailed(ctx, db.MarkAggregationImportStateFailedParams{
		TenantID: tenantID,
		ImportID: importID,
		Error:    &msg,
	}); err != nil {
		return apperr.Internal("mark import failed failed", err, map[string]string{
			"tenant_id": tenantID.String(),
			"import_id": importID.String(),
		})
	}
	return nil
}

var _ domain.ImportStateRepo = (*importStateRepo)(nil)
