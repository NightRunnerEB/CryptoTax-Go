package repository

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

type aggregatedTransactionRepo struct {
	store db.Store
}

func NewAggregatedTransactionRepo(store db.Store) domain.AggregatedTransactionRepo {
	return &aggregatedTransactionRepo{store: store}
}

func (r *aggregatedTransactionRepo) UpsertBatch(ctx context.Context, txs []domain.AggregatedTransaction) error {
	if len(txs) == 0 {
		return nil
	}
	if err := validateBatchForUpsert(txs); err != nil {
		return err
	}

	for _, tx := range txs {
		inMoney, err := moneyLegToJSON(tx.InMoney)
		if err != nil {
			return apperr.InvalidArgument("invalid in_money", err, apperr.FieldViolation{
				Field:       "in_money",
				Description: "invalid json",
			})
		}
		outMoney, err := moneyLegToJSON(tx.OutMoney)
		if err != nil {
			return apperr.InvalidArgument("invalid out_money", err, apperr.FieldViolation{
				Field:       "out_money",
				Description: "invalid json",
			})
		}
		feeMoney, err := moneyLegToJSON(tx.FeeMoney)
		if err != nil {
			return apperr.InvalidArgument("invalid fee_money", err, apperr.FieldViolation{
				Field:       "fee_money",
				Description: "invalid json",
			})
		}

		params := db.UpsertAggregatedTransactionParams{
			ID:             tx.ID,
			UserID:         tx.UserID,
			Source:         tx.Source,
			ImportID:       tx.ImportID,
			TimeUtc:        toTimestamptz(tx.TimeUTC),
			Kind:           tx.Kind,
			InMoney:        inMoney,
			OutMoney:       outMoney,
			FeeMoney:       feeMoney,
			ContractSymbol: tx.ContractSymbol,
			DerivativeKind: tx.DerivativeKind,
			PositionID:     tx.PositionID,
			OrderID:        tx.OrderID,
			TxHash:         tx.TxHash,
			Note:           tx.Note,
			TxFingerprint:  strings.TrimSpace(tx.TxFingerprint),
			CreatedAt:      toTimestamptz(tx.CreatedAt),
		}

		err = r.store.UpsertAggregatedTransaction(ctx, params)
		if err != nil {
			return apperr.Internal("upsert aggregated transaction failed", err, map[string]string{
				"user_id":   tx.UserID.String(),
				"import_id": tx.ImportID.String(),
			})
		}
	}

	return nil
}

func validateBatchForUpsert(txs []domain.AggregatedTransaction) error {
	seenFingerprints := make(map[string]string, len(txs))
	for _, tx := range txs {
		if tx.ID == uuid.Nil {
			return apperr.InvalidArgument("invalid transaction id", nil, apperr.FieldViolation{
				Field:       "id",
				Description: "required",
			})
		}
		if tx.UserID == uuid.Nil {
			return apperr.InvalidArgument("invalid user id", nil, apperr.FieldViolation{
				Field:       "user_id",
				Description: "required",
			})
		}
		if tx.ImportID == uuid.Nil {
			return apperr.InvalidArgument("invalid import id", nil, apperr.FieldViolation{
				Field:       "import_id",
				Description: "required",
			})
		}
		txFingerprint := strings.TrimSpace(tx.TxFingerprint)
		if txFingerprint == "" {
			return apperr.InvalidArgument("invalid tx_fingerprint", nil, apperr.FieldViolation{
				Field:       "tx_fingerprint",
				Description: "required",
			})
		}
		if prevTxID, exists := seenFingerprints[txFingerprint]; exists {
			return apperr.InvalidArgument("duplicate tx_fingerprint in batch", nil, apperr.FieldViolation{
				Field:       "tx_fingerprint",
				Description: "duplicate for tx_id " + tx.ID.String() + " and " + prevTxID,
			})
		}
		seenFingerprints[txFingerprint] = tx.ID.String()
	}
	return nil
}

func (r *aggregatedTransactionRepo) List(
	ctx context.Context,
	userID uuid.UUID,
	filter domain.ListTransactionsFilter,
	pageSize int32,
	cursor *domain.AggregatedTxCursor,
) ([]domain.AggregatedTransaction, bool, error) {
	params := db.ListAggregatedTransactionsParams{
		UserID:    userID,
		DateFrom:  nullableTimestamptz(filter.DateFrom),
		DateTo:    nullableTimestamptz(filter.DateTo),
		ImportID:  filter.ImportID,
		Source:    optionalText(filter.Source),
		Kind:      optionalText(filter.Kind),
		HasCursor: cursor != nil,
		PageLimit: pageSize + 1,
	}
	if cursor != nil {
		params.CursorTime = toTimestamptz(cursor.LastTimeUTC)
		params.CursorID = cursor.LastID
	}

	rows, err := r.store.ListAggregatedTransactions(ctx, params)
	if err != nil {
		return nil, false, apperr.Internal("list aggregated transactions failed", err, map[string]string{
			"user_id": userID.String(),
		})
	}

	hasMore := len(rows) > int(pageSize)
	if hasMore {
		rows = rows[:pageSize]
	}

	out := make([]domain.AggregatedTransaction, 0, len(rows))
	for _, row := range rows {
		tx, err := mapAggregatedTransactionRow(row)
		if err != nil {
			return nil, false, err
		}
		out = append(out, tx)
	}

	return out, hasMore, nil
}

func (r *aggregatedTransactionRepo) ListByRange(ctx context.Context, userID uuid.UUID, fromUTC, toUTC time.Time, limit, offset int32) (domain.AggregatedTxPage, error) {
	count, err := r.store.CountAggregatedTransactionsByRange(ctx, db.CountAggregatedTransactionsByRangeParams{
		UserID:    userID,
		TimeUtc:   toTimestamptz(fromUTC),
		TimeUtc_2: toTimestamptz(toUTC),
	})
	if err != nil {
		return domain.AggregatedTxPage{}, apperr.Internal("count aggregated transactions by range failed", err, map[string]string{
			"user_id": userID.String(),
		})
	}

	rows, err := r.store.ListAggregatedTransactionsByRange(ctx, db.ListAggregatedTransactionsByRangeParams{
		UserID:    userID,
		TimeUtc:   toTimestamptz(fromUTC),
		TimeUtc_2: toTimestamptz(toUTC),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return domain.AggregatedTxPage{}, apperr.Internal("list aggregated transactions by range failed", err, map[string]string{
			"user_id": userID.String(),
		})
	}

	out := make([]domain.AggregatedTransaction, 0, len(rows))
	for _, row := range rows {
		tx, err := mapAggregatedTransactionRow(row)
		if err != nil {
			return domain.AggregatedTxPage{}, err
		}
		out = append(out, tx)
	}

	return domain.AggregatedTxPage{Transactions: out, Total: count}, nil
}

func mapAggregatedTransactionRow(row db.AggregatedTransaction) (domain.AggregatedTransaction, error) {
	inMoney, err := moneyLegFromJSON(row.InMoney)
	if err != nil {
		return domain.AggregatedTransaction{}, apperr.Internal("invalid in_money json", err, nil)
	}
	outMoney, err := moneyLegFromJSON(row.OutMoney)
	if err != nil {
		return domain.AggregatedTransaction{}, apperr.Internal("invalid out_money json", err, nil)
	}
	feeMoney, err := moneyLegFromJSON(row.FeeMoney)
	if err != nil {
		return domain.AggregatedTransaction{}, apperr.Internal("invalid fee_money json", err, nil)
	}

	return domain.AggregatedTransaction{
		ID:             row.ID,
		UserID:         row.UserID,
		Source:         row.Source,
		ImportID:       row.ImportID,
		TimeUTC:        fromTimestamptz(row.TimeUtc),
		Kind:           row.Kind,
		InMoney:        inMoney,
		OutMoney:       outMoney,
		FeeMoney:       feeMoney,
		ContractSymbol: row.ContractSymbol,
		DerivativeKind: row.DerivativeKind,
		PositionID:     row.PositionID,
		OrderID:        row.OrderID,
		TxHash:         row.TxHash,
		Note:           row.Note,
		TxFingerprint:  row.TxFingerprint,
		CreatedAt:      fromTimestamptz(row.CreatedAt),
	}, nil
}

func nullableTimestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return toTimestamptz(*value)
}

func optionalText(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

var _ domain.AggregatedTransactionRepo = (*aggregatedTransactionRepo)(nil)
