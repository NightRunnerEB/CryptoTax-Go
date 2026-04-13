package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Lot = событие приобретения одного актива.
// Используется только для FIFO / cost basis.
type Lot struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	AcquiredAt time.Time

	Asset        string // BTC/ETH/USDT...
	QtyTotal     decimal.Decimal
	QtyRemaining decimal.Decimal
	CostFiat     decimal.Decimal

	SourceTxID uuid.UUID // AggregatedTransaction.ID
	Source     string    // Bybit/OKX...
	OrderID    *string   // если биржа дала
	TxHash     *string   // если on-chain
}
