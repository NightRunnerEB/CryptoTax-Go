package grpcserver

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	v1 "github.com/NightRunner/CryptoTax-Go/gen/price/v1"
	applogger "github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

type LegKind int

const (
	LegIn LegKind = iota + 1
	LegOut
	LegFee
)

type slot struct {
	txIdx  int
	kind   LegKind
	symbol string
	coinID string
	amount decimal.Decimal
	result **v1.FiatLeg
}

type PriceServer struct {
	v1.UnimplementedPriceServer
	resolver          domain.CoinIdResolver
	historicalPriceUC domain.HistoricalPriceUseCase
	userSymbolUC      domain.UserSymbolUseCase
}

func NewPriceServer(resolver domain.CoinIdResolver, historicalPriceUC domain.HistoricalPriceUseCase, userSymbolUC domain.UserSymbolUseCase) *PriceServer {
	return &PriceServer{
		resolver:          resolver,
		historicalPriceUC: historicalPriceUC,
		userSymbolUC:      userSymbolUC,
	}
}

func (server *PriceServer) ValuateTransactionsBatch(ctx context.Context, req *v1.ValuateTransactionsRequest) (*v1.ValuateTransactionsResponse, error) {
	start := time.Now()
	log := applogger.FromContext(ctx)
	log.Info(
		"ValuateTransactionsBatch: start",
		zap.Int("txs", len(req.Transactions)),
		zap.String("fiat", req.FiatCurrency),
	)

	resp := &v1.ValuateTransactionsResponse{
		Transactions: make([]*v1.ValuatedTx, len(req.Transactions)),
	}

	var slots []slot
	var priceKeys []domain.PriceKey

	for i, tx := range req.Transactions {
		if tx.TimeUtc == nil {
			log.Warn("ValuateTransactionsBatch: missing TimeUtc", zap.Int("tx_idx", i))
			return nil, apperr.InvalidArgument(
				"transaction time is required",
				nil,
				apperr.FieldViolation{
					Field:       "transactions[" + strconv.Itoa(i) + "].time_utc",
					Description: "required",
				},
			)
		}
		if tx.TimeUtc.AsTime().After(time.Now().UTC()) {
			return nil, apperr.InvalidArgument(
				"transaction time cannot be in the future",
				nil,
				apperr.FieldViolation{
					Field:       "transactions[" + strconv.Itoa(i) + "].time_utc",
					Description: "must not be in the future",
				},
			)
		}

		out := &v1.ValuatedTx{
			TxId: tx.TxId,
		}
		resp.Transactions[i] = out

		add := func(kind LegKind, m *v1.MoneyLeg, amountField string, result **v1.FiatLeg) error {
			if m == nil {
				return nil
			}

			if m.Amount == "" {
				return apperr.InvalidArgument(
					"amount is required",
					nil,
					apperr.FieldViolation{
						Field:       amountField,
						Description: "required",
					},
				)
			}
			amount, err := decimal.NewFromString(m.Amount)
			if err != nil {
				return apperr.InvalidArgument(
					"invalid amount",
					err,
					apperr.FieldViolation{
						Field:       amountField,
						Description: "invalid decimal",
					},
				)
			}

			// В будущем Resolve должен возвращать
			coinID, err := server.resolver.Resolve(m.Symbol)
			if err != nil {
				*result = &v1.FiatLeg{
					Error: &v1.AssetError{
						Symbol:     m.Symbol,
						Code:       v1.AssetErrorCode_ASSET_UNKNOWN,
						Candidates: nil,
					},
				}
				return nil
			}

			slots = append(slots, slot{
				txIdx:  i,
				kind:   kind,
				symbol: m.Symbol,
				coinID: coinID,
				amount: amount,
				result: result,
			})
			priceKeys = append(priceKeys, domain.PriceKey{CoinID: coinID, BucketStartUtc: tx.TimeUtc.AsTime()})
			return nil
		}
		if err := add(LegIn, tx.InMoney, "transactions["+strconv.Itoa(i)+"].in_money.amount", &out.InFiat); err != nil {
			return nil, err
		}
		if err := add(LegOut, tx.OutMoney, "transactions["+strconv.Itoa(i)+"].out_money.amount", &out.OutFiat); err != nil {
			return nil, err
		}
		if err := add(LegFee, tx.FeeMoney, "transactions["+strconv.Itoa(i)+"].fee_money.amount", &out.FeeFiat); err != nil {
			return nil, err
		}
	}

	if len(slots) == 0 {
		log.Info(
			"ValuateTransactionsBatch: no slots",
			zap.Int("txs", len(req.Transactions)),
			zap.Duration("duration", time.Since(start)),
		)
		return resp, nil
	}

	fiats, err := server.historicalPriceUC.GetHistoricalPrices(ctx, req.FiatCurrency, priceKeys)
	if err != nil {
		return nil, err
	}

	if len(fiats) != len(priceKeys) {
		return nil, apperr.Internal(
			"pricing invariant violated",
			nil,
			map[string]string{
				"got":      strconv.Itoa(len(fiats)),
				"expected": strconv.Itoa(len(priceKeys)),
			},
		)
	}

	for i, fiat := range fiats {
		s := slots[i]
		total := fiat.Mul(s.amount)
		*s.result = &v1.FiatLeg{
			Fiat: total.String(),
		}
	}

	log.Info(
		"ValuateTransactionsBatch: done",
		zap.Int("txs", len(req.Transactions)),
		zap.Duration("duration", time.Since(start)),
	)
	return resp, nil
}

func (server PriceServer) UpsertUserSymbol(ctx context.Context, req *v1.UpsertUserSymbolRequest) (*v1.UpsertUserSymbolResponse, error) {
	log := applogger.FromContext(ctx)
	userId, err := parseUUID(req.UserId)
	if err != nil {
		log.Warn("UpsertUserSymbol: invalid user ID", zap.Error(err))
		return nil, apperr.InvalidArgument(
			"invalid user id",
			err,
			apperr.FieldViolation{
				Field:       "user_id",
				Description: "invalid format",
			},
		)
	}

	userSymbol := domain.UserSymbol{
		UserID: userId,
		Source: req.Source,
		Symbol: req.Symbol,
		CoinID: req.CoinId,
	}

	if err := server.userSymbolUC.Upsert(ctx, userSymbol); err != nil {
		return nil, err
	}

	log.Info(
		"UpsertUserSymbol: upserted",
		zap.String("user_id", userId.String()),
		zap.String("source", req.Source),
		zap.String("symbol", req.Symbol),
	)
	return &v1.UpsertUserSymbolResponse{}, nil
}

func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
