package domain

import (
	"time"

	"github.com/google/uuid"
)

type LedgerAsset struct {
	Symbol string `json:"symbol"`
	Amount string `json:"amount"`
}

type LedgerTransaction struct {
	ID             uuid.UUID    `json:"id"`
	UserID         uuid.UUID    `json:"user_id"`
	Source         string       `json:"source"`
	TimeUTC        time.Time    `json:"time_utc"`
	Kind           string       `json:"kind"`
	InMoney        *LedgerAsset `json:"in_money"`
	OutMoney       *LedgerAsset `json:"out_money"`
	FeeMoney       *LedgerAsset `json:"fee_money"`
	ContractSymbol *string      `json:"contract_symbol"`
	DerivativeKind *string      `json:"derivative_kind"`
	PositionID     *string      `json:"position_id"`
	OrderID        *string      `json:"order_id"`
	TxHash         *string      `json:"tx_hash"`
	Note           *string      `json:"note"`
	ImportID       uuid.UUID    `json:"import_id"`
	TxFingerprint  string       `json:"tx_fingerprint"`
	CreatedAt      time.Time    `json:"created_at"`
}
