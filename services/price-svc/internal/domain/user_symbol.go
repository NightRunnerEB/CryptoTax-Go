package domain

import (
	"context"

	"github.com/google/uuid"
)

type UserSymbol struct {
	UserID uuid.UUID `json:"user_id"`
	Source string    `json:"source"`
	Symbol string    `json:"symbol"`
	CoinID string    `json:"coin_id"`
}

type UserSymbolUseCase interface {
	Upsert(ctx context.Context, s UserSymbol) error
	Delete(ctx context.Context, userID uuid.UUID, source, symbol string) error

	GetList(ctx context.Context, userID uuid.UUID, source string, symbols []string) ([]UserSymbol, error)
	GetListBySource(ctx context.Context, userID uuid.UUID, source string) ([]UserSymbol, error)
}

type UserSymbolRepo interface {
	Upsert(ctx context.Context, s UserSymbol) error
	Delete(ctx context.Context, userID uuid.UUID, source, symbol string) error

	GetList(ctx context.Context, userID uuid.UUID, source string, symbols []string) ([]UserSymbol, error)
	GetListBySource(ctx context.Context, userID uuid.UUID, source string) ([]UserSymbol, error)
}
