package grpcserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/NightRunner/CryptoTax-Go/gen/price/v1"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

type fakeResolver struct {
	resolveFn func(symbol string) (string, error)
}

func (f *fakeResolver) Resolve(symbol string) (string, error) {
	return f.resolveFn(symbol)
}

type fakeHistoricalUC struct {
	callCount int
	fiats     []domain.Fiat
	err       error
}

func (f *fakeHistoricalUC) GetHistoricalPrices(_ context.Context, _ string, _ []domain.PriceKey) ([]domain.Fiat, error) {
	f.callCount++
	if f.err != nil {
		return nil, f.err
	}
	return f.fiats, nil
}

type fakeTenantSymbolUC struct{}

func (f *fakeTenantSymbolUC) Upsert(context.Context, domain.TenantSymbol) error { return nil }
func (f *fakeTenantSymbolUC) Delete(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (f *fakeTenantSymbolUC) GetList(context.Context, uuid.UUID, string, []string) ([]domain.TenantSymbol, error) {
	return nil, nil
}
func (f *fakeTenantSymbolUC) GetListBySource(context.Context, uuid.UUID, string) ([]domain.TenantSymbol, error) {
	return nil, nil
}

func TestValuateTransactionsBatch_MissingTime(t *testing.T) {
	srv := NewPriceServer(
		&fakeResolver{resolveFn: func(symbol string) (string, error) { return symbol, nil }},
		&fakeHistoricalUC{},
		&fakeTenantSymbolUC{},
	)

	_, err := srv.ValuateTransactionsBatch(context.Background(), &v1.ValuateTransactionsRequest{
		FiatCurrency: "USD",
		Transactions: []*v1.TxToValuate{
			{TxId: "t1", InMoney: &v1.MoneyLeg{Symbol: "BTC", Amount: "1"}},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestValuateTransactionsBatch_InvalidAmount(t *testing.T) {
	srv := NewPriceServer(
		&fakeResolver{resolveFn: func(symbol string) (string, error) { return symbol, nil }},
		&fakeHistoricalUC{},
		&fakeTenantSymbolUC{},
	)

	_, err := srv.ValuateTransactionsBatch(context.Background(), &v1.ValuateTransactionsRequest{
		FiatCurrency: "USD",
		Transactions: []*v1.TxToValuate{
			{
				TxId:    "t1",
				TimeUtc: timestamppb.New(time.Now().UTC().Add(-time.Minute)),
				InMoney: &v1.MoneyLeg{Symbol: "BTC", Amount: "abc"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestValuateTransactionsBatch_FutureTime(t *testing.T) {
	srv := NewPriceServer(
		&fakeResolver{resolveFn: func(symbol string) (string, error) { return symbol, nil }},
		&fakeHistoricalUC{},
		&fakeTenantSymbolUC{},
	)

	_, err := srv.ValuateTransactionsBatch(context.Background(), &v1.ValuateTransactionsRequest{
		FiatCurrency: "USD",
		Transactions: []*v1.TxToValuate{
			{
				TxId:    "t1",
				TimeUtc: timestamppb.New(time.Now().UTC().Add(time.Hour)),
				InMoney: &v1.MoneyLeg{Symbol: "BTC", Amount: "1"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestValuateTransactionsBatch_UnknownSymbolSetsAssetError(t *testing.T) {
	unknownErr := apperr.UnknownSymbol("BTC", "")
	huc := &fakeHistoricalUC{}
	srv := NewPriceServer(
		&fakeResolver{resolveFn: func(symbol string) (string, error) { return "", unknownErr }},
		huc,
		&fakeTenantSymbolUC{},
	)

	resp, err := srv.ValuateTransactionsBatch(context.Background(), &v1.ValuateTransactionsRequest{
		FiatCurrency: "USD",
		Transactions: []*v1.TxToValuate{
			{
				TxId:    "t1",
				TimeUtc: timestamppb.New(time.Now().UTC().Add(-time.Minute)),
				InMoney: &v1.MoneyLeg{Symbol: "BTC", Amount: "1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if huc.callCount != 0 {
		t.Fatalf("historical use case must not be called, got %d calls", huc.callCount)
	}
	if resp.Transactions[0].InFiat == nil || resp.Transactions[0].InFiat.Error == nil {
		t.Fatalf("expected asset error in response, got %+v", resp.Transactions[0].InFiat)
	}
	if resp.Transactions[0].InFiat.Error.Code != v1.AssetErrorCode_ASSET_UNKNOWN {
		t.Fatalf("expected ASSET_UNKNOWN, got %v", resp.Transactions[0].InFiat.Error.Code)
	}
}

func TestValuateTransactionsBatch_NoLegsSkipsPricingUC(t *testing.T) {
	huc := &fakeHistoricalUC{}
	srv := NewPriceServer(
		&fakeResolver{resolveFn: func(symbol string) (string, error) { return "bitcoin", nil }},
		huc,
		&fakeTenantSymbolUC{},
	)

	resp, err := srv.ValuateTransactionsBatch(context.Background(), &v1.ValuateTransactionsRequest{
		FiatCurrency: "USD",
		Transactions: []*v1.TxToValuate{
			{
				TxId:    "t1",
				TimeUtc: timestamppb.New(time.Now().UTC().Add(-time.Minute)),
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if huc.callCount != 0 {
		t.Fatalf("pricing usecase should not be called, got %d", huc.callCount)
	}
	if len(resp.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(resp.Transactions))
	}
	if resp.Transactions[0].InFiat != nil || resp.Transactions[0].OutFiat != nil || resp.Transactions[0].FeeFiat != nil {
		t.Fatalf("expected empty fiat legs, got %+v", resp.Transactions[0])
	}
}

func TestValuateTransactionsBatch_HappyPath(t *testing.T) {
	fiat := decimal.NewFromInt(100)
	huc := &fakeHistoricalUC{fiats: []domain.Fiat{fiat}}
	srv := NewPriceServer(
		&fakeResolver{resolveFn: func(symbol string) (string, error) { return "bitcoin", nil }},
		huc,
		&fakeTenantSymbolUC{},
	)

	resp, err := srv.ValuateTransactionsBatch(context.Background(), &v1.ValuateTransactionsRequest{
		FiatCurrency: "USD",
		Transactions: []*v1.TxToValuate{
			{
				TxId:    "t1",
				TimeUtc: timestamppb.New(time.Now().UTC().Add(-time.Minute)),
				InMoney: &v1.MoneyLeg{Symbol: "BTC", Amount: "2"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Transactions[0].InFiat == nil {
		t.Fatal("expected in fiat result, got nil")
	}
	if resp.Transactions[0].InFiat.Fiat != "200" {
		t.Fatalf("expected fiat=200, got %s", resp.Transactions[0].InFiat.Fiat)
	}
}

func TestValuateTransactionsBatch_InvariantMismatch(t *testing.T) {
	huc := &fakeHistoricalUC{fiats: []domain.Fiat{}}
	srv := NewPriceServer(
		&fakeResolver{resolveFn: func(symbol string) (string, error) { return "bitcoin", nil }},
		huc,
		&fakeTenantSymbolUC{},
	)

	_, err := srv.ValuateTransactionsBatch(context.Background(), &v1.ValuateTransactionsRequest{
		FiatCurrency: "USD",
		Transactions: []*v1.TxToValuate{
			{
				TxId:    "t1",
				TimeUtc: timestamppb.New(time.Now().UTC().Add(-time.Minute)),
				InMoney: &v1.MoneyLeg{Symbol: "BTC", Amount: "2"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrInternal {
		t.Fatalf("expected INTERNAL_ERROR, got %v", err)
	}
}
