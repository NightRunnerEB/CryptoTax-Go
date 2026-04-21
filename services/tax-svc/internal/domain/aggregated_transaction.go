package domain

import (
	"time"

	"github.com/google/uuid"
)

type FiatLegCandidate struct {
	CoinID string `json:"coin_id"`
	Name   string `json:"name"`
}

type FiatLegError struct {
	Code       string             `json:"code"`
	Candidates []FiatLegCandidate `json:"candidates,omitempty"`
}

type MoneyLeg struct {
	Symbol       string `json:"symbol"`
	CryptoAmount string `json:"crypto_amount"`
	FiatAmount   string `json:"fiat_amount,omitempty"`
}

type Kind string

const (
	Spot             Kind = "Spot"
	Swap             Kind = "Swap"
	P2PBuy           Kind = "P2PBuy"
	P2PSell          Kind = "P2PSell"
	DepositCrypto    Kind = "DepositCrypto"
	WithdrawalCrypto Kind = "WithdrawalCrypto"
	DepositFiat      Kind = "DepositFiat"
	WithdrawalFiat   Kind = "WithdrawalFiat"
	TransferInternal Kind = "TransferInternal"
	Airdrop          Kind = "Airdrop"
	StakingReward    Kind = "StakingReward"
	Expense          Kind = "Expense"
	GiftIn           Kind = "GiftIn"
	GiftOut          Kind = "GiftOut"
	DerivativePnL    Kind = "DerivativePnL"
	FundingFee       Kind = "FundingFee"
	Stolen           Kind = "Stolen"
	Lost             Kind = "Lost"
	Burn             Kind = "Burn"
)

type AggregatedTransaction struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Source         string    `json:"source"`
	ImportID       uuid.UUID `json:"import_id"`
	TimeUTC        time.Time `json:"time_utc"`
	Kind           Kind      `json:"kind"`
	InMoney        *MoneyLeg `json:"in_money,omitempty"`
	OutMoney       *MoneyLeg `json:"out_money,omitempty"`
	FeeMoney       *MoneyLeg `json:"fee_money,omitempty"`
	ContractSymbol *string   `json:"contract_symbol,omitempty"`
	PositionID     *string   `json:"position_id,omitempty"`
	OrderID        *string   `json:"order_id,omitempty"`
	TxHash         *string   `json:"tx_hash,omitempty"`
}
