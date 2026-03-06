package engines

import (
	"context"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/events"
)

// BuildResult is an intermediate artifact produced by jurisdiction engine.
// It contains normalized events and allocations required by tax calculation.
type BuildResult struct {
	PolicySnapshot    domain.TaxPolicy
	Lots              []domain.Lot
	RealizationEvents []events.RealizationEvent
	RealizationLots   []events.RealizationLot
	IncomeEvents      []events.IncomeEvent
	ExpenseEvents     []events.ExpenseEvent
	ExpenseLots       []events.ExpenseLot
	MovementEvents    []events.MovementEvent
	Warnings          []string
}

// Engine is a jurisdiction-specific classifier/calculation preprocessor.
type Engine interface {
	Jurisdiction() domain.Jurisdiction
	Build(
		ctx context.Context,
		tenantID uuid.UUID,
		policy domain.TaxPolicy,
		transactions []domain.AggregatedTransaction,
	) (BuildResult, error)
}
