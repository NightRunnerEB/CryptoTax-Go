package usecase

import (
	"testing"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	"github.com/google/uuid"
)

func TestSortTransactionsDeterministic_TieBreakByTxID(t *testing.T) {
	baseTime := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)
	txs := []domain.AggregatedTransaction{
		{
			ID:            uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			TimeUTC:       baseTime,
			TxFingerprint: "b",
		},
		{
			ID:            uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			TimeUTC:       baseTime,
			TxFingerprint: "a",
		},
	}

	sortTransactionsDeterministic(txs)

	if got, want := txs[0].ID.String(), "11111111-1111-1111-1111-111111111111"; got != want {
		t.Fatalf("txs[0].id = %s, want %s", got, want)
	}
	if got, want := txs[1].ID.String(), "22222222-2222-2222-2222-222222222222"; got != want {
		t.Fatalf("txs[1].id = %s, want %s", got, want)
	}
}

func TestSortTransactionsDeterministic_TieBreakByFingerprintWhenIDsEqual(t *testing.T) {
	baseTime := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)
	txID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	txs := []domain.AggregatedTransaction{
		{
			ID:            txID,
			TimeUTC:       baseTime,
			TxFingerprint: "z",
		},
		{
			ID:            txID,
			TimeUTC:       baseTime,
			TxFingerprint: "a",
		},
	}

	sortTransactionsDeterministic(txs)

	if got, want := txs[0].TxFingerprint, "a"; got != want {
		t.Fatalf("txs[0].tx_fingerprint = %s, want %s", got, want)
	}
	if got, want := txs[1].TxFingerprint, "z"; got != want {
		t.Fatalf("txs[1].tx_fingerprint = %s, want %s", got, want)
	}
}
