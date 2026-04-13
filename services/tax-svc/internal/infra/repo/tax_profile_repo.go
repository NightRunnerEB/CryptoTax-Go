package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "github.com/NightRunner/CryptoTax-Go/services/tax-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type TaxProfileRepo struct {
	store db.Store
}

func NewTaxProfileRepo(store db.Store) *TaxProfileRepo {
	return &TaxProfileRepo{store: store}
}

func (r *TaxProfileRepo) Upsert(ctx context.Context, p domain.TaxProfile) error {
	walletsJSON, err := json.Marshal(p.Wallets)
	if err != nil {
		return apperr.Internal("marshal wallets failed", err, nil)
	}

	_, err = r.store.UpsertTaxProfile(ctx, db.UpsertTaxProfileParams{
		UserID:             p.UserID,
		Inn:                p.INN,
		LastName:           p.LastName,
		FirstName:          p.FirstName,
		MiddleName:         p.MiddleName,
		Timezone:           p.Timezone,
		Phone:              p.Phone,
		Wallets:            walletsJSON,
		TaxResidencyStatus: string(p.TaxResidencyStatus),
		TaxpayerType:       string(p.TaxPayerType),
	})
	if err != nil {
		return apperr.Internal("upsert tax profile failed", err, map[string]string{
			"user_id": p.UserID.String(),
		})
	}

	return nil
}

func (r *TaxProfileRepo) Get(ctx context.Context, userID uuid.UUID) (domain.TaxProfile, error) {
	row, err := r.store.GetTaxProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.TaxProfile{}, apperr.NotFound("tax profile not found", apperr.Resource{
				Type: "tax_profile",
				Name: userID.String(),
			}, err)
		}
		return domain.TaxProfile{}, apperr.Internal("get tax profile failed", err, map[string]string{
			"user_id": userID.String(),
		})
	}

	profile, err := mapTaxProfileRow(row)
	if err != nil {
		return domain.TaxProfile{}, err
	}
	return profile, nil
}

func (r *TaxProfileRepo) Delete(ctx context.Context, userID uuid.UUID) error {
	affected, err := r.store.DeleteTaxProfile(ctx, userID)
	if err != nil {
		return apperr.Internal("delete tax profile failed", err, map[string]string{
			"user_id": userID.String(),
		})
	}
	if affected == 0 {
		return apperr.NotFound("tax profile not found", apperr.Resource{
			Type: "tax_profile",
			Name: userID.String(),
		}, nil)
	}
	return nil
}

func mapTaxProfileRow(row db.TaxProfile) (domain.TaxProfile, error) {
	var wallets []domain.Wallet
	if err := json.Unmarshal(row.Wallets, &wallets); err != nil {
		return domain.TaxProfile{}, apperr.Internal("unmarshal wallets failed", err, nil)
	}

	return domain.TaxProfile{
		UserID:             row.UserID,
		INN:                row.Inn,
		LastName:           row.LastName,
		FirstName:          row.FirstName,
		MiddleName:         row.MiddleName,
		Timezone:           row.Timezone,
		Phone:              row.Phone,
		Wallets:            wallets,
		TaxResidencyStatus: domain.TaxResidency(row.TaxResidencyStatus),
		TaxPayerType:       domain.TaxPayerType(row.TaxpayerType),
	}, nil
}

var _ domain.TaxProfileRepo = (*TaxProfileRepo)(nil)
