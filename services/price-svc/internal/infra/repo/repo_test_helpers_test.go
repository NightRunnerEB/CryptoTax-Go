package repository

import (
	"context"

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
)

type fakeStore struct {
	deleteUserSymbolFn            func(ctx context.Context, arg db.DeleteUserSymbolParams) (int64, error)
	getFXRateFn                   func(ctx context.Context, arg db.GetFXRateParams) (db.FxRate, error)
	getHistoricalPriceFn          func(ctx context.Context, arg db.GetHistoricalPriceParams) (db.HistoricalPrice, error)
	getHistoricalPricesBatchFn    func(ctx context.Context, arg db.GetHistoricalPricesBatchParams) ([]db.GetHistoricalPricesBatchRow, error)
	getUserSymbolsFn              func(ctx context.Context, arg db.GetUserSymbolsParams) ([]db.UserSymbol, error)
	listFXRatesByFiatFn           func(ctx context.Context, fiat string) ([]db.FxRate, error)
	listUserSymbolsBySourceFn     func(ctx context.Context, arg db.ListUserSymbolsBySourceParams) ([]db.UserSymbol, error)
	upsertFXRateFn                func(ctx context.Context, arg db.UpsertFXRateParams) error
	upsertHistoricalPriceFn       func(ctx context.Context, arg db.UpsertHistoricalPriceParams) error
	upsertHistoricalPricesBatchFn func(ctx context.Context, arg db.UpsertHistoricalPricesBatchParams) error
	upsertUserSymbolFn            func(ctx context.Context, arg db.UpsertUserSymbolParams) error
}

func (f *fakeStore) DeleteUserSymbol(ctx context.Context, arg db.DeleteUserSymbolParams) (int64, error) {
	if f.deleteUserSymbolFn != nil {
		return f.deleteUserSymbolFn(ctx, arg)
	}
	return 0, nil
}

func (f *fakeStore) GetFXRate(ctx context.Context, arg db.GetFXRateParams) (db.FxRate, error) {
	if f.getFXRateFn != nil {
		return f.getFXRateFn(ctx, arg)
	}
	return db.FxRate{}, nil
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

func (f *fakeStore) GetUserSymbols(ctx context.Context, arg db.GetUserSymbolsParams) ([]db.UserSymbol, error) {
	if f.getUserSymbolsFn != nil {
		return f.getUserSymbolsFn(ctx, arg)
	}
	return nil, nil
}

func (f *fakeStore) ListFXRatesByFiat(ctx context.Context, fiat string) ([]db.FxRate, error) {
	if f.listFXRatesByFiatFn != nil {
		return f.listFXRatesByFiatFn(ctx, fiat)
	}
	return nil, nil
}

func (f *fakeStore) ListUserSymbolsBySource(ctx context.Context, arg db.ListUserSymbolsBySourceParams) ([]db.UserSymbol, error) {
	if f.listUserSymbolsBySourceFn != nil {
		return f.listUserSymbolsBySourceFn(ctx, arg)
	}
	return nil, nil
}

func (f *fakeStore) UpsertFXRate(ctx context.Context, arg db.UpsertFXRateParams) error {
	if f.upsertFXRateFn != nil {
		return f.upsertFXRateFn(ctx, arg)
	}
	return nil
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

func (f *fakeStore) UpsertUserSymbol(ctx context.Context, arg db.UpsertUserSymbolParams) error {
	if f.upsertUserSymbolFn != nil {
		return f.upsertUserSymbolFn(ctx, arg)
	}
	return nil
}
