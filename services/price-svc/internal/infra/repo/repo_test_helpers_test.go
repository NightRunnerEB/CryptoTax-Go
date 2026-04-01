package repository

import (
	"context"

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
)

type fakeStore struct {
	deleteTenantSymbolFn          func(ctx context.Context, arg db.DeleteTenantSymbolParams) (int64, error)
	getHistoricalPriceFn          func(ctx context.Context, arg db.GetHistoricalPriceParams) (db.HistoricalPrice, error)
	getHistoricalPricesBatchFn    func(ctx context.Context, arg db.GetHistoricalPricesBatchParams) ([]db.GetHistoricalPricesBatchRow, error)
	getTenantSymbolsFn            func(ctx context.Context, arg db.GetTenantSymbolsParams) ([]db.TenantSymbol, error)
	listTenantSymbolsBySourceFn   func(ctx context.Context, arg db.ListTenantSymbolsBySourceParams) ([]db.TenantSymbol, error)
	upsertHistoricalPriceFn       func(ctx context.Context, arg db.UpsertHistoricalPriceParams) error
	upsertHistoricalPricesBatchFn func(ctx context.Context, arg db.UpsertHistoricalPricesBatchParams) error
	upsertTenantSymbolFn          func(ctx context.Context, arg db.UpsertTenantSymbolParams) error
}

func (f *fakeStore) DeleteTenantSymbol(ctx context.Context, arg db.DeleteTenantSymbolParams) (int64, error) {
	if f.deleteTenantSymbolFn != nil {
		return f.deleteTenantSymbolFn(ctx, arg)
	}
	return 0, nil
}

func (f *fakeStore) GetHistoricalPrice(ctx context.Context, arg db.GetHistoricalPriceParams) (db.HistoricalPrice, error) {
	if f.getHistoricalPriceFn != nil {
		return f.getHistoricalPriceFn(ctx, arg)
	}
	return db.HistoricalPrice{}, nil
}

func (f *fakeStore) GetHistoricalPricesBatch(ctx context.Context, arg db.GetHistoricalPricesBatchParams) ([]db.GetHistoricalPricesBatchRow, error) {
	if f.getHistoricalPricesBatchFn != nil {
		return f.getHistoricalPricesBatchFn(ctx, arg)
	}
	return nil, nil
}

func (f *fakeStore) GetTenantSymbols(ctx context.Context, arg db.GetTenantSymbolsParams) ([]db.TenantSymbol, error) {
	if f.getTenantSymbolsFn != nil {
		return f.getTenantSymbolsFn(ctx, arg)
	}
	return nil, nil
}

func (f *fakeStore) ListTenantSymbolsBySource(ctx context.Context, arg db.ListTenantSymbolsBySourceParams) ([]db.TenantSymbol, error) {
	if f.listTenantSymbolsBySourceFn != nil {
		return f.listTenantSymbolsBySourceFn(ctx, arg)
	}
	return nil, nil
}

func (f *fakeStore) UpsertHistoricalPrice(ctx context.Context, arg db.UpsertHistoricalPriceParams) error {
	if f.upsertHistoricalPriceFn != nil {
		return f.upsertHistoricalPriceFn(ctx, arg)
	}
	return nil
}

func (f *fakeStore) UpsertHistoricalPricesBatch(ctx context.Context, arg db.UpsertHistoricalPricesBatchParams) error {
	if f.upsertHistoricalPricesBatchFn != nil {
		return f.upsertHistoricalPricesBatchFn(ctx, arg)
	}
	return nil
}

func (f *fakeStore) UpsertTenantSymbol(ctx context.Context, arg db.UpsertTenantSymbolParams) error {
	if f.upsertTenantSymbolFn != nil {
		return f.upsertTenantSymbolFn(ctx, arg)
	}
	return nil
}
