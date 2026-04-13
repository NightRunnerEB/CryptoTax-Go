package kz

import (
	"context"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/engines"
)

// Engine is KZ-specific normalization/classification engine.
type Engine struct{}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Jurisdiction() domain.Jurisdiction {
	return domain.JurisdictionKZ
}

func (e *Engine) Build(
	ctx context.Context,
	userID uuid.UUID,
	policy domain.TaxPolicy,
	transactions []domain.AggregatedTransaction,
) (engines.BuildResult, error) {
	_ = ctx
	_ = userID
	_ = policy
	_ = transactions

	return engines.BuildResult{}, apperr.NotImplemented("kz engine classifier is not implemented yet", nil, nil)
}

var _ engines.Engine = (*Engine)(nil)
