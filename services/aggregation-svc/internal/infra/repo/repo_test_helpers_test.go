package repository

import (
	"context"

	"github.com/google/uuid"

	db "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/db/sqlc"
)

type fakeStore struct {
	countByImportFn         func(ctx context.Context, arg db.CountAggregatedTransactionsByImportParams) (int64, error)
	countByRangeFn          func(ctx context.Context, arg db.CountAggregatedTransactionsByRangeParams) (int64, error)
	getImportStateFn        func(ctx context.Context, arg db.GetAggregationImportStateParams) (db.AggregationImportState, error)
	getTenantSettingsFn     func(ctx context.Context, tenantID uuid.UUID) (db.TenantSetting, error)
	listByImportFn          func(ctx context.Context, arg db.ListAggregatedTransactionsByImportParams) ([]db.AggregatedTransaction, error)
	listByRangeFn           func(ctx context.Context, arg db.ListAggregatedTransactionsByRangeParams) ([]db.AggregatedTransaction, error)
	markCompletedFn         func(ctx context.Context, arg db.MarkAggregationImportStateCompletedParams) error
	markFailedFn            func(ctx context.Context, arg db.MarkAggregationImportStateFailedParams) error
	updateByFingerprintFn   func(ctx context.Context, arg db.UpdateAggregatedTransactionByFingerprintParams) (int64, error)
	upsertAggregatedTxFn    func(ctx context.Context, arg db.UpsertAggregatedTransactionParams) error
	upsertProcessingStateFn func(ctx context.Context, arg db.UpsertAggregationImportStateProcessingParams) error
	upsertTenantSettingsFn  func(ctx context.Context, arg db.UpsertTenantSettingsParams) (db.TenantSetting, error)
}

func (f *fakeStore) CountAggregatedTransactionsByImport(ctx context.Context, arg db.CountAggregatedTransactionsByImportParams) (int64, error) {
	if f.countByImportFn != nil {
		return f.countByImportFn(ctx, arg)
	}
	return 0, nil
}

func (f *fakeStore) CountAggregatedTransactionsByRange(ctx context.Context, arg db.CountAggregatedTransactionsByRangeParams) (int64, error) {
	if f.countByRangeFn != nil {
		return f.countByRangeFn(ctx, arg)
	}
	return 0, nil
}

func (f *fakeStore) GetAggregationImportState(ctx context.Context, arg db.GetAggregationImportStateParams) (db.AggregationImportState, error) {
	if f.getImportStateFn != nil {
		return f.getImportStateFn(ctx, arg)
	}
	return db.AggregationImportState{}, nil
}

func (f *fakeStore) GetTenantSettings(ctx context.Context, tenantID uuid.UUID) (db.TenantSetting, error) {
	if f.getTenantSettingsFn != nil {
		return f.getTenantSettingsFn(ctx, tenantID)
	}
	return db.TenantSetting{}, nil
}

func (f *fakeStore) ListAggregatedTransactionsByImport(ctx context.Context, arg db.ListAggregatedTransactionsByImportParams) ([]db.AggregatedTransaction, error) {
	if f.listByImportFn != nil {
		return f.listByImportFn(ctx, arg)
	}
	return nil, nil
}

func (f *fakeStore) ListAggregatedTransactionsByRange(ctx context.Context, arg db.ListAggregatedTransactionsByRangeParams) ([]db.AggregatedTransaction, error) {
	if f.listByRangeFn != nil {
		return f.listByRangeFn(ctx, arg)
	}
	return nil, nil
}

func (f *fakeStore) MarkAggregationImportStateCompleted(ctx context.Context, arg db.MarkAggregationImportStateCompletedParams) error {
	if f.markCompletedFn != nil {
		return f.markCompletedFn(ctx, arg)
	}
	return nil
}

func (f *fakeStore) MarkAggregationImportStateFailed(ctx context.Context, arg db.MarkAggregationImportStateFailedParams) error {
	if f.markFailedFn != nil {
		return f.markFailedFn(ctx, arg)
	}
	return nil
}

func (f *fakeStore) UpdateAggregatedTransactionByFingerprint(ctx context.Context, arg db.UpdateAggregatedTransactionByFingerprintParams) (int64, error) {
	if f.updateByFingerprintFn != nil {
		return f.updateByFingerprintFn(ctx, arg)
	}
	return 0, nil
}

func (f *fakeStore) UpsertAggregatedTransaction(ctx context.Context, arg db.UpsertAggregatedTransactionParams) error {
	if f.upsertAggregatedTxFn != nil {
		return f.upsertAggregatedTxFn(ctx, arg)
	}
	return nil
}

func (f *fakeStore) UpsertAggregationImportStateProcessing(ctx context.Context, arg db.UpsertAggregationImportStateProcessingParams) error {
	if f.upsertProcessingStateFn != nil {
		return f.upsertProcessingStateFn(ctx, arg)
	}
	return nil
}

func (f *fakeStore) UpsertTenantSettings(ctx context.Context, arg db.UpsertTenantSettingsParams) (db.TenantSetting, error) {
	if f.upsertTenantSettingsFn != nil {
		return f.upsertTenantSettingsFn(ctx, arg)
	}
	return db.TenantSetting{}, nil
}

func strPtr(v string) *string {
	return &v
}
