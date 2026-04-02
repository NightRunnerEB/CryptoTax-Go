package ru

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/events"
)

func TestEngineBuild_SpotSell_ConsumesFIFOAndKeepsRemainder(t *testing.T) {
	engine := New()
	tenantID := uuid.New()
	now := time.Now().UTC()

	buyTx := txWithLegs(uuid.New(), tenantID, now, domain.Spot,
		leg("BTC", "1", "1000000"),
		leg("RUB", "1000000", "1000000"),
		nil,
	)
	sellTx := txWithLegs(uuid.New(), tenantID, now.Add(time.Minute), domain.Spot,
		leg("RUB", "600000", "600000"),
		leg("BTC", "0.4", "600000"),
		nil,
	)

	result, err := engine.Build(context.Background(), tenantID, domain.TaxPolicy{
		Jurisdiction:                domain.JurisdictionRU,
		CostBasisMethod:             domain.FIFO,
		TreatCryptoCryptoAsDisposal: true,
	}, []domain.AggregatedTransaction{buyTx, sellTx})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if len(result.RealizationEvents) != 1 {
		t.Fatalf("realization events length mismatch: got %d want 1", len(result.RealizationEvents))
	}
	realization := result.RealizationEvents[0]
	if !realization.ProceedsFiat.Equal(dec("600000")) {
		t.Fatalf("proceeds mismatch: got %s want 600000", realization.ProceedsFiat)
	}
	if !realization.CostBasisFiat.Equal(dec("400000")) {
		t.Fatalf("cost basis mismatch: got %s want 400000", realization.CostBasisFiat)
	}
	if len(result.RealizationLots) != 1 {
		t.Fatalf("realization lots length mismatch: got %d want 1", len(result.RealizationLots))
	}
	if !result.RealizationLots[0].Qty.Equal(dec("0.4")) {
		t.Fatalf("realization lot qty mismatch: got %s want 0.4", result.RealizationLots[0].Qty)
	}
	if !result.RealizationLots[0].CostFiat.Equal(dec("400000")) {
		t.Fatalf("realization lot cost mismatch: got %s want 400000", result.RealizationLots[0].CostFiat)
	}

	if len(result.Lots) != 1 {
		t.Fatalf("lots length mismatch: got %d want 1", len(result.Lots))
	}
	if !result.Lots[0].QtyRemaining.Equal(dec("0.6")) {
		t.Fatalf("lot qty remaining mismatch: got %s want 0.6", result.Lots[0].QtyRemaining)
	}
	if !result.Lots[0].CostFiat.Equal(dec("600000")) {
		t.Fatalf("lot cost remaining mismatch: got %s want 600000", result.Lots[0].CostFiat)
	}
}

func TestEngineBuild_SwapPolicyFalse_ReturnsNotImplemented(t *testing.T) {
	engine := New()
	tenantID := uuid.New()
	tx := txWithLegs(uuid.New(), tenantID, time.Now().UTC(), domain.Swap,
		leg("BTC", "1", "2500000"),
		leg("ETH", "10", "2500000"),
		nil,
	)

	_, err := engine.Build(context.Background(), tenantID, domain.TaxPolicy{
		Jurisdiction:                domain.JurisdictionRU,
		CostBasisMethod:             domain.FIFO,
		TreatCryptoCryptoAsDisposal: false,
	}, []domain.AggregatedTransaction{tx})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertAppErrCode(t, err, apperr.ErrNotImplemented)
}

func TestEngineBuild_SwapPolicyTrue_CreatesRealizationAndLot(t *testing.T) {
	engine := New()
	tenantID := uuid.New()
	now := time.Now().UTC()

	buyETH := txWithLegs(uuid.New(), tenantID, now, domain.Spot,
		leg("ETH", "2", "400000"),
		leg("RUB", "400000", "400000"),
		nil,
	)
	swap := txWithLegs(uuid.New(), tenantID, now.Add(time.Minute), domain.Swap,
		leg("BTC", "1", "500000"),
		leg("ETH", "2", "500000"),
		nil,
	)

	result, err := engine.Build(context.Background(), tenantID, domain.TaxPolicy{
		Jurisdiction:                domain.JurisdictionRU,
		CostBasisMethod:             domain.FIFO,
		TreatCryptoCryptoAsDisposal: true,
	}, []domain.AggregatedTransaction{buyETH, swap})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if len(result.RealizationEvents) != 1 {
		t.Fatalf("realization events length mismatch: got %d want 1", len(result.RealizationEvents))
	}
	if result.RealizationEvents[0].Kind != events.RealizationSwapOut {
		t.Fatalf("realization kind mismatch: got %s want %s", result.RealizationEvents[0].Kind, events.RealizationSwapOut)
	}
	if !result.RealizationEvents[0].ProceedsFiat.Equal(dec("500000")) {
		t.Fatalf("proceeds mismatch: got %s want 500000", result.RealizationEvents[0].ProceedsFiat)
	}
	if len(result.Lots) != 2 {
		t.Fatalf("lots length mismatch: got %d want 2", len(result.Lots))
	}
	if result.Lots[1].Asset != "BTC" {
		t.Fatalf("new lot asset mismatch: got %s want BTC", result.Lots[1].Asset)
	}
	if !result.Lots[1].CostFiat.Equal(dec("500000")) {
		t.Fatalf("new lot cost mismatch: got %s want 500000", result.Lots[1].CostFiat)
	}
}

func TestEngineBuild_Airdrop_CreatesIncomeAndLot(t *testing.T) {
	engine := New()
	tenantID := uuid.New()
	tx := txWithLegs(uuid.New(), tenantID, time.Now().UTC(), domain.Airdrop,
		leg("BTC", "0.1", "300000"),
		nil,
		nil,
	)

	result, err := engine.Build(context.Background(), tenantID, domain.TaxPolicy{
		Jurisdiction:                domain.JurisdictionRU,
		CostBasisMethod:             domain.FIFO,
		TreatCryptoCryptoAsDisposal: true,
	}, []domain.AggregatedTransaction{tx})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if len(result.IncomeEvents) != 1 {
		t.Fatalf("income events length mismatch: got %d want 1", len(result.IncomeEvents))
	}
	if result.IncomeEvents[0].Kind != events.IncomeAirdrop {
		t.Fatalf("income kind mismatch: got %s want %s", result.IncomeEvents[0].Kind, events.IncomeAirdrop)
	}
	if len(result.Lots) != 1 {
		t.Fatalf("lots length mismatch: got %d want 1", len(result.Lots))
	}
	if result.Lots[0].Asset != "BTC" {
		t.Fatalf("lot asset mismatch: got %s want BTC", result.Lots[0].Asset)
	}
	if !result.Lots[0].CostFiat.Equal(dec("300000")) {
		t.Fatalf("lot cost mismatch: got %s want 300000", result.Lots[0].CostFiat)
	}
}

func TestEngineBuild_ExpenseCrypto_CreatesExpenseAllocation(t *testing.T) {
	engine := New()
	tenantID := uuid.New()
	now := time.Now().UTC()

	buyTx := txWithLegs(uuid.New(), tenantID, now, domain.Spot,
		leg("BTC", "1", "1000000"),
		leg("RUB", "1000000", "1000000"),
		nil,
	)
	expenseTx := txWithLegs(uuid.New(), tenantID, now.Add(time.Minute), domain.Expense,
		nil,
		leg("BTC", "0.25", "50000"),
		nil,
	)

	result, err := engine.Build(context.Background(), tenantID, domain.TaxPolicy{
		Jurisdiction:                domain.JurisdictionRU,
		CostBasisMethod:             domain.FIFO,
		TreatCryptoCryptoAsDisposal: true,
	}, []domain.AggregatedTransaction{buyTx, expenseTx})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if len(result.ExpenseEvents) != 1 {
		t.Fatalf("expense events length mismatch: got %d want 1", len(result.ExpenseEvents))
	}
	if result.ExpenseEvents[0].Kind != events.ExpenseManual {
		t.Fatalf("expense kind mismatch: got %s want %s", result.ExpenseEvents[0].Kind, events.ExpenseManual)
	}
	if len(result.ExpenseLots) != 1 {
		t.Fatalf("expense lots length mismatch: got %d want 1", len(result.ExpenseLots))
	}
	if !result.ExpenseLots[0].Qty.Equal(dec("0.25")) {
		t.Fatalf("expense lot qty mismatch: got %s want 0.25", result.ExpenseLots[0].Qty)
	}
	if !result.ExpenseLots[0].CostFiat.Equal(dec("250000")) {
		t.Fatalf("expense lot cost mismatch: got %s want 250000", result.ExpenseLots[0].CostFiat)
	}
	if !result.Lots[0].QtyRemaining.Equal(dec("0.75")) {
		t.Fatalf("lot qty remaining mismatch: got %s want 0.75", result.Lots[0].QtyRemaining)
	}
}

func TestEngineBuild_NegativeInventory_ReturnsDomainError(t *testing.T) {
	engine := New()
	tenantID := uuid.New()
	sellTx := txWithLegs(uuid.New(), tenantID, time.Now().UTC(), domain.Spot,
		leg("RUB", "100000", "100000"),
		leg("BTC", "0.1", "100000"),
		nil,
	)

	_, err := engine.Build(context.Background(), tenantID, domain.TaxPolicy{
		Jurisdiction:                domain.JurisdictionRU,
		CostBasisMethod:             domain.FIFO,
		TreatCryptoCryptoAsDisposal: true,
	}, []domain.AggregatedTransaction{sellTx})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertAppErrCode(t, err, apperr.ErrNegativeInventory)
}

func TestEngineBuild_SameTimestamp_UsesTxIDTieBreaker(t *testing.T) {
	engine := New()
	tenantID := uuid.New()
	at := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)

	firstID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	sellID := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	// Deliberately pass out of order: second first, then first.
	buySecond := txWithLegs(secondID, tenantID, at, domain.Spot,
		leg("BTC", "1", "300"),
		leg("RUB", "300", "300"),
		nil,
	)
	buyFirst := txWithLegs(firstID, tenantID, at, domain.Spot,
		leg("BTC", "1", "100"),
		leg("RUB", "100", "100"),
		nil,
	)
	sell := txWithLegs(sellID, tenantID, at.Add(time.Minute), domain.Spot,
		leg("RUB", "600", "600"),
		leg("BTC", "1.5", "600"),
		nil,
	)

	result, err := engine.Build(context.Background(), tenantID, domain.TaxPolicy{
		Jurisdiction:                domain.JurisdictionRU,
		CostBasisMethod:             domain.FIFO,
		TreatCryptoCryptoAsDisposal: true,
	}, []domain.AggregatedTransaction{buySecond, buyFirst, sell})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if len(result.RealizationEvents) != 1 {
		t.Fatalf("realization events length mismatch: got %d want 1", len(result.RealizationEvents))
	}
	if !result.RealizationEvents[0].CostBasisFiat.Equal(dec("250")) {
		t.Fatalf("cost basis mismatch (tie-breaker failed): got %s want 250", result.RealizationEvents[0].CostBasisFiat)
	}
}

func txWithLegs(id, tenantID uuid.UUID, when time.Time, kind domain.Kind, in, out, fee *domain.MoneyLeg) domain.AggregatedTransaction {
	return domain.AggregatedTransaction{
		ID:       id,
		TenantID: tenantID,
		Source:   "MEXC",
		ImportID: uuid.New(),
		TimeUTC:  when,
		Kind:     kind,
		InMoney:  in,
		OutMoney: out,
		FeeMoney: fee,
	}
}

func leg(symbol, qty, fiat string) *domain.MoneyLeg {
	return &domain.MoneyLeg{
		Symbol:       symbol,
		CryptoAmount: qty,
		FiatAmount:   fiat,
	}
}

func dec(v string) decimal.Decimal {
	d, err := decimal.NewFromString(v)
	if err != nil {
		panic(err)
	}
	return d
}

func assertAppErrCode(t *testing.T, err error, want apperr.ErrorCode) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected app error, got %T: %v", err, err)
	}
	if ae.Code != want {
		t.Fatalf("error code mismatch: got %s want %s", ae.Code, want)
	}
}
