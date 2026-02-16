package domain

import (
	"time"

	"github.com/google/uuid"
)

type MoneyLeg struct {
	Symbol       string        `json:"symbol"`
	CryptoAmount string        `json:"crypto_amount"`
	FiatAmount   *string       `json:"fiat_amount,omitempty"`
	Error        *FiatLegError `json:"error,omitempty"`
}

type FiatLegCandidate struct {
	CoinID string `json:"coin_id"`
	Name   string `json:"name"`
}

type FiatLegError struct {
	Code       string             `json:"code"`
	Candidates []FiatLegCandidate `json:"candidates,omitempty"`
}

type AggregatedTransaction struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Source         string    `json:"source"`
	ImportID       uuid.UUID `json:"import_id"`
	TimeUTC        time.Time `json:"time_utc"`
	Kind           string    `json:"kind"`
	InMoney        *MoneyLeg `json:"in_money,omitempty"`
	OutMoney       *MoneyLeg `json:"out_money,omitempty"`
	FeeMoney       *MoneyLeg `json:"fee_money,omitempty"`
	ContractSymbol *string   `json:"contract_symbol,omitempty"`
	DerivativeKind *string   `json:"derivative_kind,omitempty"`
	PositionID     *string   `json:"position_id,omitempty"`
	OrderID        *string   `json:"order_id,omitempty"`
	TxHash         *string   `json:"tx_hash,omitempty"`
	Note           *string   `json:"note,omitempty"`
	TxFingerprint  string    `json:"tx_fingerprint"`
	CreatedAt      time.Time `json:"created_at"`
}
