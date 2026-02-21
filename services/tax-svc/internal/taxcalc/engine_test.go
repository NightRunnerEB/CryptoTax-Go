package taxcalc

import (
	"errors"
	"testing"
	"time"

	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

func TestFIFO_SingleBuySell(t *testing.T) {
	engine := NewEngine(Policy{FailOnNegativeInventory: true})
	ts := time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)

	err := engine.ApplyAcquisition("BTC", MustParse("1"), MustParse("1000000"), ts, "buy-1", "", "BUY", nil)
	if err != nil {
		t.Fatalf("ApplyAcquisition() error = %v", err)
	}

	disposal, err := engine.ApplyDisposition("BTC", MustParse("0.4"), MustParse("600000"), Zero(), "sell-1")
	if err != nil {
		t.Fatalf("ApplyDisposition() error = %v", err)
	}

	assertAmount(t, disposal.CostFiat, "400000")
	assertAmount(t, disposal.GainFiat, "200000")
}

func TestFIFO_SplitAcrossTwoLots(t *testing.T) {
	engine := NewEngine(Policy{FailOnNegativeInventory: true})
	ts := time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)

	if err := engine.ApplyAcquisition("BTC", MustParse("0.5"), MustParse("500000"), ts, "buy-1", "", "BUY", nil); err != nil {
		t.Fatalf("ApplyAcquisition(buy-1) error = %v", err)
	}
	if err := engine.ApplyAcquisition("BTC", MustParse("0.5"), MustParse("700000"), ts.Add(time.Minute), "buy-2", "", "BUY", nil); err != nil {
		t.Fatalf("ApplyAcquisition(buy-2) error = %v", err)
	}

	disposal, err := engine.ApplyDisposition("BTC", MustParse("0.6"), MustParse("900000"), Zero(), "sell-1")
	if err != nil {
		t.Fatalf("ApplyDisposition() error = %v", err)
	}

	if got, want := len(disposal.Lines), 2; got != want {
		t.Fatalf("len(disposal.Lines) = %d, want %d", got, want)
	}
	assertAmount(t, disposal.Lines[0].QtyDisposed, "0.5")
	assertAmount(t, disposal.Lines[1].QtyDisposed, "0.1")
	assertAmount(t, disposal.CostFiat, "640000")
	assertAmount(t, disposal.GainFiat, "260000")
}

func TestFIFO_FiatFeesAffectCostAndProceeds(t *testing.T) {
	engine := NewEngine(Policy{FailOnNegativeInventory: true})
	ts := time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)

	// Buy with fiat fee included in acquisition cost.
	if err := engine.ApplyAcquisition("BTC", MustParse("1"), MustParse("1010"), ts, "buy-1", "", "BUY", nil); err != nil {
		t.Fatalf("ApplyAcquisition() error = %v", err)
	}

	// Sell with fiat fee reducing net result.
	disposal, err := engine.ApplyDisposition("BTC", MustParse("1"), MustParse("2000"), MustParse("20"), "sell-1")
	if err != nil {
		t.Fatalf("ApplyDisposition() error = %v", err)
	}

	assertAmount(t, disposal.CostFiat, "1010")
	assertAmount(t, disposal.FeesFiat, "20")
	assertAmount(t, disposal.GainFiat, "970")
}

func TestFIFO_CryptoFeeCreatesSeparateDisposition(t *testing.T) {
	engine := NewEngine(Policy{FailOnNegativeInventory: true})
	ts := time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)

	if err := engine.ApplyAcquisition("BTC", MustParse("1"), MustParse("1000"), ts, "buy-1", "", "BUY", nil); err != nil {
		t.Fatalf("ApplyAcquisition() error = %v", err)
	}

	main, err := engine.ApplyDisposition("BTC", MustParse("0.5"), MustParse("800"), Zero(), "sell-1")
	if err != nil {
		t.Fatalf("ApplyDisposition(main) error = %v", err)
	}
	assertAmount(t, main.CostFiat, "500")

	fee, err := engine.ApplyDisposition("BTC", MustParse("0.1"), Zero(), Zero(), "fee-1")
	if err != nil {
		t.Fatalf("ApplyDisposition(fee) error = %v", err)
	}
	assertAmount(t, fee.CostFiat, "100")

	remaining, err := engine.ApplyDisposition("BTC", MustParse("0.4"), Zero(), Zero(), "rest-1")
	if err != nil {
		t.Fatalf("ApplyDisposition(remaining) error = %v", err)
	}
	assertAmount(t, remaining.CostFiat, "400")
}

func TestFIFO_AirdropLikeAcquisitionCostUsedOnSale(t *testing.T) {
	engine := NewEngine(Policy{FailOnNegativeInventory: true})
	ts := time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)

	// Income lot: 2 ETH with total fiat basis 200.
	if err := engine.ApplyAcquisition("ETH", MustParse("2"), MustParse("200"), ts, "airdrop-1", "", "INCOME", nil); err != nil {
		t.Fatalf("ApplyAcquisition() error = %v", err)
	}

	disposal, err := engine.ApplyDisposition("ETH", MustParse("1"), MustParse("150"), Zero(), "sell-1")
	if err != nil {
		t.Fatalf("ApplyDisposition() error = %v", err)
	}

	assertAmount(t, disposal.CostFiat, "100")
	assertAmount(t, disposal.GainFiat, "50")
}

func TestFIFO_NegativeInventoryFails(t *testing.T) {
	engine := NewEngine(Policy{FailOnNegativeInventory: true})

	_, err := engine.ApplyDisposition("BTC", MustParse("1"), MustParse("100"), Zero(), "sell-1")
	if err == nil {
		t.Fatal("ApplyDisposition() error = nil, want negative inventory error")
	}

	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T, want *apperr.Error", err)
	}
	if appErr.Code != apperr.ErrNegativeInventory {
		t.Fatalf("error code = %s, want %s", appErr.Code, apperr.ErrNegativeInventory)
	}
}

func assertAmount(t *testing.T, got Amount, want string) {
	t.Helper()
	if got.String() != want {
		t.Fatalf("amount = %s, want %s", got.String(), want)
	}
}
