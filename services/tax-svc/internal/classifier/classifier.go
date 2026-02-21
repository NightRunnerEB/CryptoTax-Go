package classifier

import (
	"fmt"
	"strings"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

const (
	txKindSpot             = "Spot"
	txKindSwap             = "Swap"
	txKindDepositCrypto    = "DepositCrypto"
	txKindWithdrawalCrypto = "WithdrawalCrypto"
	txKindDepositFiat      = "DepositFiat"
	txKindWithdrawalFiat   = "WithdrawalFiat"
	txKindTransferInternal = "TransferInternal"
	txKindAirdrop          = "Airdrop"
	txKindStakingReward    = "StakingReward"
	txKindExpense          = "Expense"
	txKindGiftIn           = "GiftIn"
	txKindGiftOut          = "GiftOut"
	txKindDerivativePnL    = "DerivativePnL"
	txKindFundingFee       = "FundingFee"
	txKindStolen           = "Stolen"
	txKindLost             = "Lost"
	txKindBurn             = "Burn"
)

var fiatSymbols = map[string]struct{}{
	"USD": {}, "EUR": {}, "RUB": {}, "GBP": {}, "JPY": {}, "CNY": {}, "CHF": {}, "KZT": {},
}

func Classify(tx domain.AggregatedTransaction, policy Policy) ([]domain.DatasetEvent, error) {
	meta := txMeta(tx)
	base := domain.DatasetEvent{
		TxID:          tx.ID.String(),
		TimeUTC:       tx.TimeUTC,
		Source:        tx.Source,
		Kind:          tx.Kind,
		TxFingerprint: tx.TxFingerprint,
		Meta:          meta,
	}

	switch tx.Kind {
	case txKindSpot:
		return classifySpot(base, tx, policy)
	case txKindSwap:
		return classifySwap(base, tx, policy)
	case txKindDepositCrypto, txKindWithdrawalCrypto, txKindTransferInternal, txKindDepositFiat, txKindWithdrawalFiat:
		ev := base
		ev.EventType = domain.EventTransfer
		fillFromLeg(&ev, firstNonNil(tx.InMoney, tx.OutMoney))
		return []domain.DatasetEvent{ev}, nil
	case txKindAirdrop, txKindStakingReward:
		ev := base
		ev.EventType = domain.EventIncome
		fillFromLeg(&ev, tx.InMoney)
		if err := requireFiat(tx.ID.String(), tx.Kind, "in_money", tx.InMoney); err != nil {
			return nil, err
		}
		ev.FiatAmount = tx.InMoney.FiatAmount
		return []domain.DatasetEvent{ev}, nil
	case txKindExpense, txKindFundingFee:
		ev := base
		ev.EventType = domain.EventExpense
		leg := firstNonNil(tx.FeeMoney, tx.OutMoney)
		fillFromLeg(&ev, leg)
		if err := requireFiat(tx.ID.String(), tx.Kind, "fee_or_out", leg); err != nil {
			return nil, err
		}
		ev.FiatAmount = leg.FiatAmount
		return []domain.DatasetEvent{ev}, nil
	case txKindDerivativePnL:
		if tx.InMoney != nil {
			ev := base
			ev.EventType = domain.EventIncome
			fillFromLeg(&ev, tx.InMoney)
			if err := requireFiat(tx.ID.String(), tx.Kind, "in_money", tx.InMoney); err != nil {
				return nil, err
			}
			ev.FiatAmount = tx.InMoney.FiatAmount
			return []domain.DatasetEvent{ev}, nil
		}
		ev := base
		ev.EventType = domain.EventExpense
		fillFromLeg(&ev, firstNonNil(tx.OutMoney, tx.FeeMoney))
		leg := firstNonNil(tx.OutMoney, tx.FeeMoney)
		if err := requireFiat(tx.ID.String(), tx.Kind, "out_or_fee", leg); err != nil {
			return nil, err
		}
		ev.FiatAmount = leg.FiatAmount
		return []domain.DatasetEvent{ev}, nil
	case txKindGiftIn:
		ev := base
		ev.EventType = domain.EventGiftIn
		fillFromLeg(&ev, tx.InMoney)
		return []domain.DatasetEvent{ev}, nil
	case txKindGiftOut:
		ev := base
		ev.EventType = domain.EventGiftOut
		fillFromLeg(&ev, tx.OutMoney)
		return []domain.DatasetEvent{ev}, nil
	case txKindStolen, txKindLost, txKindBurn:
		ev := base
		ev.EventType = domain.EventLoss
		fillFromLeg(&ev, firstNonNil(tx.OutMoney, tx.InMoney))
		return []domain.DatasetEvent{ev}, nil
	default:
		ev := base
		ev.EventType = domain.EventTransfer
		fillFromLeg(&ev, firstNonNil(tx.InMoney, tx.OutMoney))
		return []domain.DatasetEvent{ev}, nil
	}
}

func classifySpot(base domain.DatasetEvent, tx domain.AggregatedTransaction, policy Policy) ([]domain.DatasetEvent, error) {
	inLeg := tx.InMoney
	outLeg := tx.OutMoney
	if inLeg == nil && outLeg == nil {
		ev := base
		ev.EventType = domain.EventTransfer
		return []domain.DatasetEvent{ev}, nil
	}

	inFiat := isFiatSymbol(inLeg)
	outFiat := isFiatSymbol(outLeg)

	if outFiat && inLeg != nil {
		ev := base
		ev.EventType = domain.EventAcquisition
		fillFromLeg(&ev, inLeg)
		if err := requireFiat(tx.ID.String(), tx.Kind, "out_money", outLeg); err != nil {
			return nil, err
		}
		ev.FiatAmount = outLeg.FiatAmount
		return []domain.DatasetEvent{ev}, nil
	}

	if inFiat && outLeg != nil {
		ev := base
		ev.EventType = domain.EventDisposition
		fillFromLeg(&ev, outLeg)
		if err := requireFiat(tx.ID.String(), tx.Kind, "in_money", inLeg); err != nil {
			return nil, err
		}
		ev.FiatAmount = inLeg.FiatAmount
		return []domain.DatasetEvent{ev}, nil
	}

	// Fallback for unknown symbol classification.
	return classifySwap(base, tx, policy)
}

func classifySwap(base domain.DatasetEvent, tx domain.AggregatedTransaction, policy Policy) ([]domain.DatasetEvent, error) {
	if !policy.TreatCryptoToCryptoAsDisposition {
		ev := base
		ev.EventType = domain.EventTransfer
		fillFromLeg(&ev, firstNonNil(tx.InMoney, tx.OutMoney))
		return []domain.DatasetEvent{ev}, nil
	}

	out := base
	out.EventType = domain.EventDisposition
	fillFromLeg(&out, tx.OutMoney)
	if err := requireFiat(tx.ID.String(), tx.Kind, "out_money", tx.OutMoney); err != nil {
		return nil, err
	}
	out.FiatAmount = tx.OutMoney.FiatAmount

	in := base
	in.EventType = domain.EventAcquisition
	fillFromLeg(&in, tx.InMoney)
	if err := requireFiat(tx.ID.String(), tx.Kind, "in_money", tx.InMoney); err != nil {
		return nil, err
	}
	in.FiatAmount = tx.InMoney.FiatAmount

	return []domain.DatasetEvent{out, in}, nil
}

func txMeta(tx domain.AggregatedTransaction) map[string]string {
	meta := make(map[string]string)
	put := func(key string, value *string) {
		if value == nil {
			return
		}
		v := strings.TrimSpace(*value)
		if v == "" {
			return
		}
		meta[key] = v
	}
	put("contract_symbol", tx.ContractSymbol)
	put("derivative_kind", tx.DerivativeKind)
	put("position_id", tx.PositionID)
	put("order_id", tx.OrderID)
	put("tx_hash", tx.TxHash)
	put("note", tx.Note)
	if len(meta) == 0 {
		return nil
	}
	return meta
}

func requireFiat(txID, kind, field string, leg *domain.MoneyLeg) error {
	if leg == nil {
		return apperr.NeedsPriceResolution(
			"needs price resolution",
			nil,
			map[string]string{"tx_id": txID, "kind": kind, "field": field, "reason": "leg is nil"},
		)
	}
	if leg.Error != nil {
		return apperr.NeedsPriceResolution(
			"needs price resolution",
			nil,
			map[string]string{"tx_id": txID, "kind": kind, "field": field, "reason": "price leg contains error"},
		)
	}
	if leg.FiatAmount == nil || strings.TrimSpace(*leg.FiatAmount) == "" {
		return apperr.NeedsPriceResolution(
			"needs price resolution",
			nil,
			map[string]string{"tx_id": txID, "kind": kind, "field": field, "reason": "fiat amount is empty"},
		)
	}
	return nil
}

func fillFromLeg(event *domain.DatasetEvent, leg *domain.MoneyLeg) {
	if event == nil || leg == nil {
		return
	}
	event.AssetSymbol = strings.TrimSpace(leg.Symbol)
	event.CryptoAmount = strings.TrimSpace(leg.CryptoAmount)
	event.FiatAmount = leg.FiatAmount
}

func firstNonNil(legs ...*domain.MoneyLeg) *domain.MoneyLeg {
	for _, leg := range legs {
		if leg != nil {
			return leg
		}
	}
	return nil
}

func isFiatSymbol(leg *domain.MoneyLeg) bool {
	if leg == nil {
		return false
	}
	s := strings.ToUpper(strings.TrimSpace(leg.Symbol))
	_, ok := fiatSymbols[s]
	return ok
}

func formatNeedsPriceErr(tx domain.AggregatedTransaction, field string) error {
	return apperr.NeedsPriceResolution(
		"needs price resolution",
		fmt.Errorf("missing fiat value for %s", field),
		map[string]string{
			"tx_id": tx.ID.String(),
			"kind":  tx.Kind,
			"field": field,
		},
	)
}
