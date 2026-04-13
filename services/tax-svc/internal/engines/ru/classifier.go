package ru

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/events"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/engines"
)

var fiatLikeSymbols = map[string]struct{}{
	"RUB": {}, "USD": {}, "EUR": {}, "KZT": {}, "GBP": {}, "JPY": {}, "CHF": {}, "CNY": {},
	"USDT": {}, "USDC": {}, "BUSD": {}, "FDUSD": {}, "TUSD": {}, "DAI": {}, "USDP": {}, "USDD": {},
}

type lotAllocation struct {
	lotID    uuid.UUID
	qty      decimal.Decimal
	costFiat decimal.Decimal
}

type parsedLeg struct {
	symbol string
	qty    decimal.Decimal
	fiat   decimal.Decimal
}

// Engine is RU-specific normalization/classification engine.
type Engine struct{}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Jurisdiction() domain.Jurisdiction {
	return domain.JurisdictionRU
}

func (e *Engine) Build(
	ctx context.Context,
	userID uuid.UUID,
	policy domain.TaxPolicy,
	transactions []domain.AggregatedTransaction,
) (engines.BuildResult, error) {
	method := policy.CostBasisMethod

	switch method {
	case domain.FIFO, domain.LIFO, domain.AVG:
	default:
		return engines.BuildResult{}, apperr.InvalidArgument("unsupported cost basis method", nil, apperr.FieldViolation{
			Field:       "tax_policy.cost_basis_method",
			Description: "must be FIFO, LIFO or AVG",
		})
	}

	sorted := make([]domain.AggregatedTransaction, len(transactions))
	copy(sorted, transactions)
	sort.Slice(sorted, func(i, j int) bool {
		ti := sorted[i].TimeUTC.UTC()
		tj := sorted[j].TimeUTC.UTC()
		if ti.Equal(tj) {
			return sorted[i].ID.String() < sorted[j].ID.String()
		}
		return ti.Before(tj)
	})

	result := engines.BuildResult{
		PolicySnapshot: policy,
	}
	inventory := make(map[string][]int)

	for _, tx := range sorted {
		if err := ctx.Err(); err != nil {
			return engines.BuildResult{}, err
		}
		if tx.ID == uuid.Nil {
			return engines.BuildResult{}, apperr.InvalidTxShape("transaction id is required", nil, map[string]string{
				"kind": string(tx.Kind),
			})
		}
		if tx.UserID != uuid.Nil && tx.UserID != userID {
			return engines.BuildResult{}, apperr.InvalidTxShape("user mismatch in transaction", nil, map[string]string{
				"tx_id":       tx.ID.String(),
				"user_id":     tx.UserID.String(),
				"expected_id": userID.String(),
			})
		}

		if err := e.applyTransaction(&result, inventory, userID, method, policy, tx); err != nil {
			return engines.BuildResult{}, err
		}
	}

	return result, nil
}

func (e *Engine) applyTransaction(
	result *engines.BuildResult,
	inventory map[string][]int,
	userID uuid.UUID,
	method domain.CostBasisMethod,
	policy domain.TaxPolicy,
	tx domain.AggregatedTransaction,
) error {
	inLeg, hasIn, err := parseLeg(tx, "in_money", tx.InMoney)
	if err != nil {
		return err
	}
	outLeg, hasOut, err := parseLeg(tx, "out_money", tx.OutMoney)
	if err != nil {
		return err
	}
	feeLeg, hasFee, err := parseLeg(tx, "fee_money", tx.FeeMoney)
	if err != nil {
		return err
	}

	switch tx.Kind {
	case domain.Spot:
		return e.handleSpot(result, inventory, userID, method, policy, tx, inLeg, hasIn, outLeg, hasOut, feeLeg, hasFee)
	case domain.Swap:
		return e.handleSwap(result, inventory, userID, method, policy, tx, inLeg, hasIn, outLeg, hasOut, feeLeg, hasFee)
	case domain.DepositCrypto:
		return addMovement(result, userID, tx, events.MovementIn, inLeg, hasIn)
	case domain.WithdrawalCrypto:
		if err := addMovement(result, userID, tx, events.MovementOut, outLeg, hasOut); err != nil {
			return err
		}
		if hasFee {
			return addExpense(result, inventory, method, userID, tx, feeLeg, events.ExpenseNetworkFee)
		}
		return nil
	case domain.DepositFiat:
		return addMovement(result, userID, tx, events.MovementIn, inLeg, hasIn)
	case domain.WithdrawalFiat:
		return addMovement(result, userID, tx, events.MovementOut, outLeg, hasOut)
	case domain.TransferInternal:
		return addInternalMovement(result, userID, tx, inLeg, hasIn, outLeg, hasOut)
	case domain.Airdrop:
		return addIncomeWithLot(result, inventory, userID, tx, inLeg, hasIn, events.IncomeAirdrop)
	case domain.StakingReward:
		return addIncomeWithLot(result, inventory, userID, tx, inLeg, hasIn, events.IncomeStakingReward)
	case domain.Expense:
		return handleManualExpense(result, inventory, method, userID, tx, outLeg, hasOut, feeLeg, hasFee)
	case domain.GiftIn:
		return addIncomeWithLot(result, inventory, userID, tx, inLeg, hasIn, events.IncomeGiftIn)
	case domain.GiftOut:
		return addRealization(result, inventory, method, userID, tx, outLeg, hasOut, decimal.Zero, events.RealizationGift)
	case domain.DerivativePnL:
		return handleDerivativePnL(result, inventory, method, userID, tx, inLeg, hasIn, outLeg, hasOut)
	case domain.FundingFee:
		return handleFundingFee(result, inventory, method, userID, tx, feeLeg, hasFee, outLeg, hasOut)
	case domain.Stolen:
		return addRealization(result, inventory, method, userID, tx, outLeg, hasOut, decimal.Zero, events.RealizationStolen)
	case domain.Lost:
		return addRealization(result, inventory, method, userID, tx, outLeg, hasOut, decimal.Zero, events.RealizationLost)
	case domain.Burn:
		return addRealization(result, inventory, method, userID, tx, outLeg, hasOut, decimal.Zero, events.RealizationBurn)
	default:
		return apperr.UnsupportedKind("unsupported tx kind", nil, map[string]string{
			"tx_id": tx.ID.String(),
			"kind":  string(tx.Kind),
		})
	}
}

func (e *Engine) handleSpot(
	result *engines.BuildResult,
	inventory map[string][]int,
	userID uuid.UUID,
	method domain.CostBasisMethod,
	policy domain.TaxPolicy,
	tx domain.AggregatedTransaction,
	inLeg parsedLeg,
	hasIn bool,
	outLeg parsedLeg,
	hasOut bool,
	feeLeg parsedLeg,
	hasFee bool,
) error {
	if !hasIn || !hasOut {
		return invalidShape(tx, "spot requires both in_money and out_money")
	}

	inFiat := isFiatLike(inLeg.symbol)
	outFiat := isFiatLike(outLeg.symbol)

	switch {
	case !outFiat && inFiat:
		proceeds := inLeg.fiat
		if err := addRealization(result, inventory, method, userID, tx, outLeg, true, proceeds, events.RealizationSellFiat); err != nil {
			return err
		}
	case outFiat && !inFiat:
		cost := inLeg.fiat
		addLot(result, inventory, userID, tx, inLeg.symbol, inLeg.qty, cost)
	case !outFiat && !inFiat:
		if !policy.TreatCryptoCryptoAsDisposal {
			return apperr.NotImplemented("spot crypto-to-crypto without disposal is not supported in MVP", nil, map[string]string{
				"tx_id": tx.ID.String(),
			})
		}
		proceeds := inLeg.fiat
		if err := addRealization(result, inventory, method, userID, tx, outLeg, true, proceeds, events.RealizationSwapOut); err != nil {
			return err
		}
		cost := inLeg.fiat
		addLot(result, inventory, userID, tx, inLeg.symbol, inLeg.qty, cost)
	default:
		result.Warnings = append(result.Warnings, fmt.Sprintf("spot tx %s is fiat-to-fiat and was skipped", tx.ID.String()))
	}

	if hasFee {
		if err := addExpense(result, inventory, method, userID, tx, feeLeg, events.ExpenseTradeFee); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) handleSwap(
	result *engines.BuildResult,
	inventory map[string][]int,
	userID uuid.UUID,
	method domain.CostBasisMethod,
	policy domain.TaxPolicy,
	tx domain.AggregatedTransaction,
	inLeg parsedLeg,
	hasIn bool,
	outLeg parsedLeg,
	hasOut bool,
	feeLeg parsedLeg,
	hasFee bool,
) error {
	if !hasIn || !hasOut {
		return invalidShape(tx, "swap requires both in_money and out_money")
	}
	if isFiatLike(inLeg.symbol) || isFiatLike(outLeg.symbol) {
		return invalidShape(tx, "swap must be crypto-to-crypto")
	}
	if !policy.TreatCryptoCryptoAsDisposal {
		return apperr.NotImplemented("swap without disposal is not supported in MVP", nil, map[string]string{
			"tx_id": tx.ID.String(),
		})
	}

	proceeds := inLeg.fiat
	if err := addRealization(result, inventory, method, userID, tx, outLeg, true, proceeds, events.RealizationSwapOut); err != nil {
		return err
	}

	cost := inLeg.fiat
	addLot(result, inventory, userID, tx, inLeg.symbol, inLeg.qty, cost)

	if hasFee {
		if err := addExpense(result, inventory, method, userID, tx, feeLeg, events.ExpenseTradeFee); err != nil {
			return err
		}
	}
	return nil
}

func addIncomeWithLot(
	result *engines.BuildResult,
	inventory map[string][]int,
	userID uuid.UUID,
	tx domain.AggregatedTransaction,
	inLeg parsedLeg,
	hasIn bool,
	kind events.IncomeKind,
) error {
	if !hasIn {
		return invalidShape(tx, "income transaction requires in_money")
	}
	amountFiat := inLeg.fiat
	event := events.IncomeEvent{
		ID:         uuid.New(),
		UserID:     userID,
		OccurredAt: tx.TimeUTC.UTC(),
		AmountFiat: amountFiat,
		Asset:      inLeg.symbol,
		Qty:        inLeg.qty,
		Kind:       kind,
		Evidence:   evidenceFromTx(tx),
	}
	result.IncomeEvents = append(result.IncomeEvents, event)
	if !isFiatLike(inLeg.symbol) {
		addLot(result, inventory, userID, tx, inLeg.symbol, inLeg.qty, amountFiat)
	}
	return nil
}

func handleManualExpense(
	result *engines.BuildResult,
	inventory map[string][]int,
	method domain.CostBasisMethod,
	userID uuid.UUID,
	tx domain.AggregatedTransaction,
	outLeg parsedLeg,
	hasOut bool,
	feeLeg parsedLeg,
	hasFee bool,
) error {
	if hasOut {
		return addExpense(result, inventory, method, userID, tx, outLeg, events.ExpenseManual)
	}
	if hasFee {
		return addExpense(result, inventory, method, userID, tx, feeLeg, events.ExpenseManual)
	}
	return invalidShape(tx, "expense requires out_money or fee_money")
}

func handleDerivativePnL(
	result *engines.BuildResult,
	inventory map[string][]int,
	method domain.CostBasisMethod,
	userID uuid.UUID,
	tx domain.AggregatedTransaction,
	inLeg parsedLeg,
	hasIn bool,
	outLeg parsedLeg,
	hasOut bool,
) error {
	handled := false

	if hasIn && inLeg.qty.GreaterThan(decimal.Zero) {
		amountFiat := inLeg.fiat
		result.IncomeEvents = append(result.IncomeEvents, events.IncomeEvent{
			ID:         uuid.New(),
			UserID:     userID,
			OccurredAt: tx.TimeUTC.UTC(),
			AmountFiat: amountFiat,
			Asset:      inLeg.symbol,
			Qty:        inLeg.qty,
			Kind:       events.IncomeDerivativePnL,
			Evidence:   evidenceFromTx(tx),
		})
		if !isFiatLike(inLeg.symbol) {
			addLot(result, inventory, userID, tx, inLeg.symbol, inLeg.qty, amountFiat)
		}
		handled = true
	}

	if hasOut && outLeg.qty.GreaterThan(decimal.Zero) {
		if err := addExpense(result, inventory, method, userID, tx, outLeg, events.ExpenseDerivativeLoss); err != nil {
			return err
		}
		handled = true
	}

	if !handled {
		return invalidShape(tx, "derivative pnl requires in_money or out_money")
	}
	return nil
}

func handleFundingFee(
	result *engines.BuildResult,
	inventory map[string][]int,
	method domain.CostBasisMethod,
	userID uuid.UUID,
	tx domain.AggregatedTransaction,
	feeLeg parsedLeg,
	hasFee bool,
	outLeg parsedLeg,
	hasOut bool,
) error {
	if hasFee {
		return addExpense(result, inventory, method, userID, tx, feeLeg, events.ExpenseFundingFee)
	}
	if hasOut {
		return addExpense(result, inventory, method, userID, tx, outLeg, events.ExpenseFundingFee)
	}
	return invalidShape(tx, "funding fee requires fee_money or out_money")
}

func addMovement(
	result *engines.BuildResult,
	userID uuid.UUID,
	tx domain.AggregatedTransaction,
	direction events.MovementDirection,
	leg parsedLeg,
	ok bool,
) error {
	if !ok {
		return invalidShape(tx, "movement leg is required")
	}
	result.MovementEvents = append(result.MovementEvents, events.MovementEvent{
		ID:         uuid.New(),
		UserID:     userID,
		OccurredAt: tx.TimeUTC.UTC(),
		Asset:      leg.symbol,
		Qty:        leg.qty,
		Direction:  direction,
		Evidence:   evidenceFromTx(tx),
	})
	return nil
}

func addInternalMovement(
	result *engines.BuildResult,
	userID uuid.UUID,
	tx domain.AggregatedTransaction,
	inLeg parsedLeg,
	hasIn bool,
	outLeg parsedLeg,
	hasOut bool,
) error {
	switch {
	case hasOut:
		return addMovement(result, userID, tx, events.MovementInternal, outLeg, true)
	case hasIn:
		return addMovement(result, userID, tx, events.MovementInternal, inLeg, true)
	default:
		return invalidShape(tx, "transfer internal requires in_money or out_money")
	}
}

func addRealization(
	result *engines.BuildResult,
	inventory map[string][]int,
	method domain.CostBasisMethod,
	userID uuid.UUID,
	tx domain.AggregatedTransaction,
	leg parsedLeg,
	ok bool,
	proceedsFiat decimal.Decimal,
	kind events.RealizationKind,
) error {
	if !ok {
		return invalidShape(tx, "realization leg is required")
	}
	if isFiatLike(leg.symbol) {
		return invalidShape(tx, "realization asset must be crypto")
	}
	if !leg.qty.GreaterThan(decimal.Zero) {
		return invalidShape(tx, "realization qty must be > 0")
	}

	allocations, totalCost, err := consumeLots(result, inventory, method, leg.symbol, leg.qty)
	if err != nil {
		return apperr.NegativeInventory("insufficient inventory for realization", err, map[string]string{
			"tx_id":  tx.ID.String(),
			"asset":  leg.symbol,
			"qty":    leg.qty.String(),
			"method": string(method),
		})
	}

	realizationID := uuid.New()
	result.RealizationEvents = append(result.RealizationEvents, events.RealizationEvent{
		ID:            realizationID,
		UserID:        userID,
		OccurredAt:    tx.TimeUTC.UTC(),
		Asset:         leg.symbol,
		Qty:           leg.qty,
		ProceedsFiat:  proceedsFiat,
		CostBasisFiat: totalCost,
		Kind:          kind,
		Evidence:      evidenceFromTx(tx),
	})

	for _, allocation := range allocations {
		result.RealizationLots = append(result.RealizationLots, events.RealizationLot{
			RealizationID: realizationID,
			LotID:         allocation.lotID,
			Qty:           allocation.qty,
			CostFiat:      allocation.costFiat,
		})
	}
	return nil
}

func addExpense(
	result *engines.BuildResult,
	inventory map[string][]int,
	method domain.CostBasisMethod,
	userID uuid.UUID,
	tx domain.AggregatedTransaction,
	leg parsedLeg,
	kind events.ExpenseKind,
) error {
	amountFiat := leg.fiat
	if !leg.qty.GreaterThan(decimal.Zero) {
		return invalidShape(tx, "expense qty must be > 0")
	}

	expenseID := uuid.New()
	result.ExpenseEvents = append(result.ExpenseEvents, events.ExpenseEvent{
		ID:         expenseID,
		UserID:     userID,
		OccurredAt: tx.TimeUTC.UTC(),
		AmountFiat: amountFiat,
		Asset:      leg.symbol,
		Qty:        leg.qty,
		Kind:       kind,
		Evidence:   evidenceFromTx(tx),
	})

	if isFiatLike(leg.symbol) {
		return nil
	}

	allocations, _, err := consumeLots(result, inventory, method, leg.symbol, leg.qty)
	if err != nil {
		return apperr.NegativeInventory("insufficient inventory for expense", err, map[string]string{
			"tx_id":  tx.ID.String(),
			"asset":  leg.symbol,
			"qty":    leg.qty.String(),
			"method": string(method),
		})
	}
	for _, allocation := range allocations {
		result.ExpenseLots = append(result.ExpenseLots, events.ExpenseLot{
			ExpenseID: expenseID,
			LotID:     allocation.lotID,
			Qty:       allocation.qty,
			CostFiat:  allocation.costFiat,
		})
	}
	return nil
}

func addLot(
	result *engines.BuildResult,
	inventory map[string][]int,
	userID uuid.UUID,
	tx domain.AggregatedTransaction,
	asset string,
	qty decimal.Decimal,
	costFiat decimal.Decimal,
) {
	lot := domain.Lot{
		ID:           uuid.New(),
		UserID:       userID,
		AcquiredAt:   tx.TimeUTC.UTC(),
		Asset:        normalizeSymbol(asset),
		QtyTotal:     qty,
		QtyRemaining: qty,
		CostFiat:     costFiat,
		SourceTxID:   tx.ID,
		Source:       tx.Source,
		OrderID:      tx.OrderID,
		TxHash:       tx.TxHash,
	}
	result.Lots = append(result.Lots, lot)
	idx := len(result.Lots) - 1
	inventory[lot.Asset] = append(inventory[lot.Asset], idx)
}

func consumeLots(
	result *engines.BuildResult,
	inventory map[string][]int,
	method domain.CostBasisMethod,
	asset string,
	qty decimal.Decimal,
) ([]lotAllocation, decimal.Decimal, error) {
	if !qty.GreaterThan(decimal.Zero) {
		return nil, decimal.Zero, fmt.Errorf("qty must be > 0")
	}

	asset = normalizeSymbol(asset)
	idxs := inventory[asset]
	active := make([]int, 0, len(idxs))
	for _, idx := range idxs {
		lot := result.Lots[idx]
		if lot.QtyRemaining.GreaterThan(decimal.Zero) {
			active = append(active, idx)
		}
	}
	if len(active) == 0 {
		return nil, decimal.Zero, fmt.Errorf("no active lots")
	}

	switch method {
	case domain.FIFO:
		return consumeSequential(result, active, qty, false)
	case domain.LIFO:
		return consumeSequential(result, active, qty, true)
	case domain.AVG:
		return consumeAverage(result, active, qty)
	default:
		return nil, decimal.Zero, fmt.Errorf("unsupported method: %s", method)
	}
}

func consumeSequential(
	result *engines.BuildResult,
	active []int,
	qty decimal.Decimal,
	reverse bool,
) ([]lotAllocation, decimal.Decimal, error) {
	allocations := make([]lotAllocation, 0, len(active))
	remaining := qty
	totalCost := decimal.Zero

	iterate := func(pos int) int {
		if reverse {
			return len(active) - 1 - pos
		}
		return pos
	}

	for pos := 0; pos < len(active) && remaining.GreaterThan(decimal.Zero); pos++ {
		idx := active[iterate(pos)]
		lot := &result.Lots[idx]
		if !lot.QtyRemaining.GreaterThan(decimal.Zero) {
			continue
		}
		take := minDecimal(remaining, lot.QtyRemaining)
		cost := lot.CostFiat.Mul(take).Div(lot.QtyRemaining)

		lot.QtyRemaining = lot.QtyRemaining.Sub(take)
		lot.CostFiat = lot.CostFiat.Sub(cost)
		clipLotValues(lot)

		remaining = remaining.Sub(take)
		totalCost = totalCost.Add(cost)
		allocations = append(allocations, lotAllocation{
			lotID:    lot.ID,
			qty:      take,
			costFiat: cost,
		})
	}

	if remaining.GreaterThan(decimal.Zero) {
		return nil, decimal.Zero, fmt.Errorf("not enough qty, missing=%s", remaining.String())
	}
	return allocations, totalCost, nil
}

func consumeAverage(
	result *engines.BuildResult,
	active []int,
	qty decimal.Decimal,
) ([]lotAllocation, decimal.Decimal, error) {
	totalQty := decimal.Zero
	totalCost := decimal.Zero
	for _, idx := range active {
		lot := result.Lots[idx]
		totalQty = totalQty.Add(lot.QtyRemaining)
		totalCost = totalCost.Add(lot.CostFiat)
	}
	if totalQty.LessThan(qty) {
		return nil, decimal.Zero, fmt.Errorf("not enough qty, available=%s", totalQty.String())
	}
	if totalQty.Equal(decimal.Zero) {
		return nil, decimal.Zero, fmt.Errorf("empty inventory")
	}

	ratio := qty.Div(totalQty)
	targetCost := totalCost.Mul(ratio)
	remainingQty := qty
	remainingCost := targetCost
	allocations := make([]lotAllocation, 0, len(active))

	for i, idx := range active {
		lot := &result.Lots[idx]
		var take decimal.Decimal
		var cost decimal.Decimal
		last := i == len(active)-1
		if last {
			take = remainingQty
			cost = remainingCost
		} else {
			take = lot.QtyRemaining.Mul(ratio)
			take = minDecimal(take, remainingQty)
			if lot.QtyRemaining.Equal(decimal.Zero) {
				cost = decimal.Zero
			} else {
				cost = lot.CostFiat.Mul(take).Div(lot.QtyRemaining)
			}
			cost = minDecimal(cost, remainingCost)
		}
		if !take.GreaterThan(decimal.Zero) {
			continue
		}

		lot.QtyRemaining = lot.QtyRemaining.Sub(take)
		lot.CostFiat = lot.CostFiat.Sub(cost)
		clipLotValues(lot)

		remainingQty = remainingQty.Sub(take)
		remainingCost = remainingCost.Sub(cost)
		allocations = append(allocations, lotAllocation{
			lotID:    lot.ID,
			qty:      take,
			costFiat: cost,
		})
	}

	if remainingQty.GreaterThan(decimal.Zero) {
		return nil, decimal.Zero, fmt.Errorf("not enough qty after avg allocation, missing=%s", remainingQty.String())
	}
	return allocations, targetCost.Sub(remainingCost), nil
}

func clipLotValues(lot *domain.Lot) {
	if lot.QtyRemaining.IsNegative() {
		lot.QtyRemaining = decimal.Zero
	}
	if lot.CostFiat.IsNegative() {
		lot.CostFiat = decimal.Zero
	}
}

func parseLeg(tx domain.AggregatedTransaction, legName string, leg *domain.MoneyLeg) (parsedLeg, bool, error) {
	if leg == nil {
		return parsedLeg{}, false, nil
	}

	symbol := normalizeSymbol(leg.Symbol)
	if symbol == "" {
		return parsedLeg{}, false, invalidShape(tx, legName+" symbol is required")
	}

	qtyRaw := strings.TrimSpace(leg.CryptoAmount)
	if qtyRaw == "" {
		return parsedLeg{}, false, invalidShape(tx, legName+" crypto_amount is required")
	}
	qty, err := decimal.NewFromString(qtyRaw)
	if err != nil {
		return parsedLeg{}, false, invalidShape(tx, legName+" crypto_amount is invalid")
	}
	if !qty.GreaterThan(decimal.Zero) {
		return parsedLeg{}, false, invalidShape(tx, legName+" crypto_amount must be > 0")
	}

	fiatRaw := strings.TrimSpace(leg.FiatAmount)
	if fiatRaw == "" {
		return parsedLeg{}, false, invalidShape(tx, legName+" fiat_amount is required")
	}
	fiat, err := decimal.NewFromString(fiatRaw)
	if err != nil {
		return parsedLeg{}, false, invalidShape(tx, legName+" fiat_amount is invalid")
	}
	if fiat.IsNegative() {
		return parsedLeg{}, false, invalidShape(tx, legName+" fiat_amount must be >= 0")
	}

	return parsedLeg{
		symbol: symbol,
		qty:    qty,
		fiat:   fiat,
	}, true, nil
}

func evidenceFromTx(tx domain.AggregatedTransaction) events.Evidence {
	return events.Evidence{
		SourceTxID: tx.ID,
		Source:     tx.Source,
		OrderID:    tx.OrderID,
		TxHash:     tx.TxHash,
	}
}

func invalidShape(tx domain.AggregatedTransaction, msg string) error {
	return apperr.InvalidTxShape(msg, nil, map[string]string{
		"tx_id": tx.ID.String(),
		"kind":  string(tx.Kind),
	})
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}

func isFiatLike(symbol string) bool {
	_, ok := fiatLikeSymbols[normalizeSymbol(symbol)]
	return ok
}

func minDecimal(a, b decimal.Decimal) decimal.Decimal {
	if a.LessThan(b) {
		return a
	}
	return b
}

var _ engines.Engine = (*Engine)(nil)
