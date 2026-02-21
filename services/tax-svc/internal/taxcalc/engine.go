package taxcalc

import (
	"strings"
	"time"

	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type Engine struct {
	policy Policy
	lots   map[string][]Lot
}

func NewEngine(policy Policy) *Engine {
	return &Engine{
		policy: policy,
		lots:   make(map[string][]Lot),
	}
}

func (e *Engine) ApplyAcquisition(
	asset string,
	qty Amount,
	costFiat Amount,
	acquiredAt time.Time,
	sourceTxID string,
	sourceFingerprint string,
	acquisitionType string,
	meta map[string]string,
) error {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	if asset == "" {
		return apperr.InvalidTxShape("acquisition asset is empty", nil, map[string]string{
			"source_tx_id": sourceTxID,
		})
	}
	if qty.Cmp(Zero()) <= 0 {
		return apperr.InvalidTxShape("acquisition qty must be positive", nil, map[string]string{
			"source_tx_id": sourceTxID,
			"asset_symbol": asset,
		})
	}

	lot := Lot{
		AssetSymbol:            asset,
		AcquiredAt:             acquiredAt.UTC(),
		QtyRemaining:           qty,
		CostTotalFiatRemaining: costFiat,
		SourceTxID:             sourceTxID,
		SourceTxFingerprint:    sourceFingerprint,
		AcquisitionType:        acquisitionType,
		Meta:                   meta,
	}
	e.lots[asset] = append(e.lots[asset], lot)
	return nil
}

func (e *Engine) ApplyDisposition(
	asset string,
	qtyToDispose Amount,
	proceedsFiat Amount,
	feesFiat Amount,
	disposalTxID string,
) (DisposalResult, error) {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	if asset == "" {
		return DisposalResult{}, apperr.InvalidTxShape("disposition asset is empty", nil, map[string]string{
			"disposal_tx_id": disposalTxID,
		})
	}
	if qtyToDispose.Cmp(Zero()) <= 0 {
		return DisposalResult{}, apperr.InvalidTxShape("disposition qty must be positive", nil, map[string]string{
			"disposal_tx_id": disposalTxID,
			"asset_symbol":   asset,
		})
	}

	originalQty := qtyToDispose
	remaining := qtyToDispose

	result := DisposalResult{
		ProceedsFiat: proceedsFiat,
		FeesFiat:     feesFiat,
	}

	queue := e.lots[asset]
	for remaining.Cmp(Zero()) > 0 {
		if len(queue) == 0 {
			if e.policy.FailOnNegativeInventory {
				return DisposalResult{}, apperr.NegativeInventory(
					"negative inventory",
					nil,
					map[string]string{
						"asset_symbol": asset,
						"tx_id":        disposalTxID,
						"qty_missing":  remaining.String(),
					},
				)
			}

			// Fallback mode: synthesize zero-cost inventory to keep pipeline moving.
			synthCost := Zero()
			synthProceeds := proceedsFiat.Mul(remaining).Div(originalQty)
			synthFees := feesFiat.Mul(remaining).Div(originalQty)
			result.CostFiat = result.CostFiat.Add(synthCost)
			result.GainFiat = result.GainFiat.Add(synthProceeds.Sub(synthCost).Sub(synthFees))
			result.Lines = append(result.Lines, RealizationLine{
				DisposalTxID:      disposalTxID,
				AssetSymbol:       asset,
				QtyDisposed:       remaining,
				ProceedsFiatAlloc: synthProceeds,
				CostFiatAlloc:     synthCost,
				FeesFiatAlloc:     synthFees,
				GainFiatAlloc:     synthProceeds.Sub(synthCost).Sub(synthFees),
				LotAcquiredAt:     time.Time{},
				LotSourceTxID:     "synthetic-zero-cost",
			})
			remaining = Zero()
			break
		}

		lot := queue[0]
		if lot.QtyRemaining.Cmp(Zero()) <= 0 {
			queue = queue[1:]
			continue
		}

		take := Min(remaining, lot.QtyRemaining)
		costAlloc := lot.CostTotalFiatRemaining.Mul(take).Div(lot.QtyRemaining)
		proceedsAlloc := proceedsFiat.Mul(take).Div(originalQty)
		feesAlloc := feesFiat.Mul(take).Div(originalQty)
		gainAlloc := proceedsAlloc.Sub(costAlloc).Sub(feesAlloc)

		result.CostFiat = result.CostFiat.Add(costAlloc)
		result.GainFiat = result.GainFiat.Add(gainAlloc)
		result.Lines = append(result.Lines, RealizationLine{
			DisposalTxID:      disposalTxID,
			AssetSymbol:       asset,
			QtyDisposed:       take,
			ProceedsFiatAlloc: proceedsAlloc,
			CostFiatAlloc:     costAlloc,
			FeesFiatAlloc:     feesAlloc,
			GainFiatAlloc:     gainAlloc,
			LotAcquiredAt:     lot.AcquiredAt,
			LotSourceTxID:     lot.SourceTxID,
		})

		lot.QtyRemaining = lot.QtyRemaining.Sub(take)
		lot.CostTotalFiatRemaining = lot.CostTotalFiatRemaining.Sub(costAlloc)
		queue[0] = lot

		remaining = remaining.Sub(take)
		if queue[0].QtyRemaining.Cmp(Zero()) == 0 {
			queue = queue[1:]
		}
	}

	e.lots[asset] = queue
	return result, nil
}

func (e *Engine) TransferCostBasis(
	outAsset string,
	outQty Amount,
	inAsset string,
	inQty Amount,
	txID string,
	txFingerprint string,
	txTime time.Time,
	meta map[string]string,
) error {
	disposal, err := e.ApplyDisposition(outAsset, outQty, Zero(), Zero(), txID)
	if err != nil {
		return err
	}
	return e.ApplyAcquisition(
		inAsset,
		inQty,
		disposal.CostFiat,
		txTime,
		txID,
		txFingerprint,
		"SWAP_IN",
		meta,
	)
}

