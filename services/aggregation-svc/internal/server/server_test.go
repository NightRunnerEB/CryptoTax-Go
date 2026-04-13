package grpcserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	aggregationv1 "github.com/NightRunner/CryptoTax-Go/gen/aggregation/v1"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/mocks"
)

func TestListTransactionsByRange_SucceedsWithoutTenantHeader(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aggUC := mocks.NewMockAggregationUseCase(ctrl)
	s := NewAggregationServer(aggUC, nil)

	tenantID := uuid.New()
	fromUTC := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toUTC := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	aggUC.EXPECT().
		ListTransactionsByRange(gomock.Any(), tenantID, fromUTC, toUTC, int32(100), int32(0), "").
		Return(domain.AggregatedTxPage{}, nil)

	_, err := s.ListTransactionsByRange(context.Background(), &aggregationv1.ListTransactionsByRangeRequest{
		TenantId: tenantID.String(),
		FromUtc:  timestamppb.New(fromUTC),
		ToUtc:    timestamppb.New(toUTC),
		Limit:    100,
		Offset:   0,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestListTransactionsByRange_InvalidTenantID(t *testing.T) {
	t.Parallel()

	s := NewAggregationServer(nil, nil)
	_, err := s.ListTransactionsByRange(context.Background(), &aggregationv1.ListTransactionsByRangeRequest{
		TenantId: "bad-uuid",
		FromUtc:  timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		ToUtc:    timestamppb.New(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServerErrorCode(t, err, apperr.ErrInvalidArgument)
}

func TestUpsertTenantSettings_MissingTenantHeader(t *testing.T) {
	t.Parallel()

	s := NewAggregationServer(nil, nil)

	_, err := s.UpsertTenantSettings(context.Background(), &aggregationv1.UpsertTenantSettingsRequest{
		FiatCurrency: "USD",
		Timezone:     "UTC",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServerErrorCode(t, err, apperr.ErrInvalidArgument)
}

func TestListSupportedFiatCurrencies_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	settingsUC := mocks.NewMockTenantSettingsUseCase(ctrl)
	s := NewAggregationServer(nil, settingsUC)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(headerTenantID, uuid.NewString()))

	settingsUC.EXPECT().
		ListSupportedFiatCurrencies(gomock.Any()).
		Return([]domain.SupportedFiatCurrency{
			{Code: "USD", DisplayName: "US Dollar"},
			{Code: "RUB", DisplayName: "Russian Ruble"},
		}, nil)

	resp, err := s.ListSupportedFiatCurrencies(ctx, &aggregationv1.ListSupportedFiatCurrenciesRequest{})
	if err != nil {
		t.Fatalf("ListSupportedFiatCurrencies returned error: %v", err)
	}
	if len(resp.GetCurrencies()) != 2 {
		t.Fatalf("unexpected currencies count: %d", len(resp.GetCurrencies()))
	}
	if resp.GetCurrencies()[0].GetCode() != "USD" || resp.GetCurrencies()[1].GetCode() != "RUB" {
		t.Fatalf("unexpected currencies: %+v", resp.GetCurrencies())
	}
}

func TestListTransactionsByImport_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aggUC := mocks.NewMockAggregationUseCase(ctrl)
	s := NewAggregationServer(aggUC, nil)

	tenantID := uuid.New()
	importID := uuid.New()
	txID := uuid.New()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(headerTenantID, tenantID.String()))

	aggUC.EXPECT().
		ListTransactionsByImport(gomock.Any(), tenantID, importID, int32(10), int32(0)).
		Return(domain.AggregatedTxPage{
			Transactions: []domain.AggregatedTransaction{
				{
					ID:            txID,
					TenantID:      tenantID,
					ImportID:      importID,
					Source:        "MEXC",
					TimeUTC:       time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
					Kind:          "Spot",
					TxFingerprint: "fp-1",
				},
			},
			Total: 1,
		}, nil)

	resp, err := s.ListTransactionsByImport(ctx, &aggregationv1.ListTransactionsByImportRequest{
		ImportId: importID.String(),
		Limit:    10,
		Offset:   0,
	})
	if err != nil {
		t.Fatalf("ListTransactionsByImport returned error: %v", err)
	}
	if resp.GetTotal() != 1 || len(resp.GetTransactions()) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.GetTransactions()[0].GetTxId() != txID.String() {
		t.Fatalf("unexpected tx id: %s", resp.GetTransactions()[0].GetTxId())
	}
}

func assertServerErrorCode(t *testing.T, err error, code apperr.ErrorCode) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected *apperr.Error, got %T (%v)", err, err)
	}
	if ae.Code != code {
		t.Fatalf("unexpected code: got=%s want=%s", ae.Code, code)
	}
}
