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

func TestListTransactionsByRange_SucceedsWithoutUserHeader(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aggUC := mocks.NewMockAggregationUseCase(ctrl)
	s := NewAggregationServer(aggUC, nil)

	userID := uuid.New()
	fromUTC := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toUTC := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	aggUC.EXPECT().
		ListTransactionsByRange(gomock.Any(), userID, fromUTC, toUTC, int32(100), int32(0), "").
		Return(domain.AggregatedTxPage{}, nil)

	_, err := s.ListTransactionsByRange(context.Background(), &aggregationv1.ListTransactionsByRangeRequest{
		UserId:  userID.String(),
		FromUtc: timestamppb.New(fromUTC),
		ToUtc:   timestamppb.New(toUTC),
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestListTransactions_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aggUC := mocks.NewMockAggregationUseCase(ctrl)
	s := NewAggregationServer(aggUC, nil)

	userID := uuid.New()
	importID := uuid.New()
	txID := uuid.New()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(headerUserID, userID.String()))

	aggUC.EXPECT().
		ListTransactions(
			gomock.Any(),
			userID,
			gomock.Any(),
			int32(30),
			"",
			"USD",
		).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, filter domain.ListTransactionsFilter, _ int32, _ string, _ string) (domain.AggregatedTxCursorPage, error) {
			if filter.ImportID == nil || *filter.ImportID != importID {
				t.Fatalf("unexpected import filter: %+v", filter.ImportID)
			}
			return domain.AggregatedTxCursorPage{
				Items: []domain.AggregatedTransaction{
					{
						ID:            txID,
						UserID:        userID,
						ImportID:      importID,
						Source:        "MEXC",
						TimeUTC:       time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
						Kind:          "spot",
						TxFingerprint: "fp-1",
					},
				},
				NextPageToken: "next-token",
			}, nil
		})

	resp, err := s.ListTransactions(ctx, &aggregationv1.ListTransactionsRequest{
		ImportId:   importID.String(),
		PageSize:   30,
		TargetFiat: "USD",
	})
	if err != nil {
		t.Fatalf("ListTransactions returned error: %v", err)
	}
	if resp.GetNextPageToken() != "next-token" {
		t.Fatalf("unexpected next_page_token: %s", resp.GetNextPageToken())
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].GetTxId() != txID.String() {
		t.Fatalf("unexpected items: %+v", resp.GetItems())
	}
}

func TestListTransactions_InvalidImportID(t *testing.T) {
	t.Parallel()

	s := NewAggregationServer(nil, nil)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(headerUserID, uuid.NewString()))

	_, err := s.ListTransactions(ctx, &aggregationv1.ListTransactionsRequest{
		ImportId: "bad-uuid",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServerErrorCode(t, err, apperr.ErrInvalidArgument)
}

func TestListTransactionsByRange_InvalidUserID(t *testing.T) {
	t.Parallel()

	s := NewAggregationServer(nil, nil)
	_, err := s.ListTransactionsByRange(context.Background(), &aggregationv1.ListTransactionsByRangeRequest{
		UserId:  "bad-uuid",
		FromUtc: timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		ToUtc:   timestamppb.New(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServerErrorCode(t, err, apperr.ErrInvalidArgument)
}

func TestUpsertUserSettings_MissingUserHeader(t *testing.T) {
	t.Parallel()

	s := NewAggregationServer(nil, nil)

	_, err := s.UpsertUserSettings(context.Background(), &aggregationv1.UpsertUserSettingsRequest{
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

	settingsUC := mocks.NewMockUserSettingsUseCase(ctrl)
	s := NewAggregationServer(nil, settingsUC)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(headerUserID, uuid.NewString()))

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
