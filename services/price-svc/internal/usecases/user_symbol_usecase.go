package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
)

type userSymbolUC struct {
	userSymbolRepository domain.UserSymbolRepo
	contextTimeout       time.Duration
}

func NewUserSymbolUC(userSymbolRepository domain.UserSymbolRepo, timeout time.Duration) domain.UserSymbolUseCase {
	return &userSymbolUC{
		userSymbolRepository: userSymbolRepository,
		contextTimeout:       timeout,
	}
}

func (u *userSymbolUC) Upsert(ctx context.Context, s domain.UserSymbol) error {
	// Implementation goes here
	return nil
}

func (u *userSymbolUC) Delete(ctx context.Context, userID uuid.UUID, source, symbol string) error {
	// Implementation goes here
	return nil
}

func (u *userSymbolUC) GetList(ctx context.Context, userID uuid.UUID, source string, symbols []string) ([]domain.UserSymbol, error) {
	// Implementation goes here
	return nil, nil
}

func (u *userSymbolUC) GetListBySource(ctx context.Context, userID uuid.UUID, source string) ([]domain.UserSymbol, error) {
	// Implementation goes here
	return nil, nil
}
