package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/mocks"
)

func TestNewHistoricalPriceUC_NotNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uc := NewHistoricalPriceUC(
		mocks.NewMockHistoricalPriceRepo(ctrl),
		mocks.NewMockFXProvider(ctrl),
		nil,
		5*time.Second,
	)
	if uc == nil {
		t.Fatal("expected non-nil usecase")
	}
}

func TestNewTenantSymbolUC_NotNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uc := NewTenantSymbolUC(mocks.NewMockTenantSymbolRepo(ctrl), 5*time.Second)
	if uc == nil {
		t.Fatal("expected non-nil usecase")
	}
}

func TestTenantSymbolUC_StubMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uc := NewTenantSymbolUC(mocks.NewMockTenantSymbolRepo(ctrl), 5*time.Second)
	ctx := context.Background()

	err := uc.Upsert(ctx, domain.TenantSymbol{
		TenantID: uuid.New(),
		Source:   "MEXC",
		Symbol:   "BTC",
		CoinID:   "bitcoin",
	})
	if err != nil {
		t.Fatalf("Upsert() unexpected error: %v", err)
	}

	err = uc.Delete(ctx, uuid.New(), "MEXC", "BTC")
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	list, err := uc.GetList(ctx, uuid.New(), "MEXC", []string{"BTC"})
	if err != nil {
		t.Fatalf("GetList() unexpected error: %v", err)
	}
	if list != nil {
		t.Fatalf("GetList() expected nil list, got %v", list)
	}

	bySource, err := uc.GetListBySource(ctx, uuid.New(), "MEXC")
	if err != nil {
		t.Fatalf("GetListBySource() unexpected error: %v", err)
	}
	if bySource != nil {
		t.Fatalf("GetListBySource() expected nil list, got %v", bySource)
	}
}
