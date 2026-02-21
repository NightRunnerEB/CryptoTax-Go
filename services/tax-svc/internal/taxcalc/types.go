package taxcalc

import "time"

type Policy struct {
	TreatSwapAsDisposition      bool
	TreatCryptoFeeAsDisposition bool
	IncludeIncomeEvents         bool
	AllowLossEventsDeduction    bool
	FailOnNegativeInventory     bool
	FailOnMissingFiat           bool
}

type Lot struct {
	AssetSymbol             string
	AcquiredAt              time.Time
	QtyRemaining            Amount
	CostTotalFiatRemaining  Amount
	SourceTxID              string
	SourceTxFingerprint     string
	AcquisitionType         string
	Meta                    map[string]string
}

type RealizationLine struct {
	DisposalTxID      string
	AssetSymbol       string
	QtyDisposed       Amount
	ProceedsFiatAlloc Amount
	CostFiatAlloc     Amount
	FeesFiatAlloc     Amount
	GainFiatAlloc     Amount
	LotAcquiredAt     time.Time
	LotSourceTxID     string
}

type DisposalResult struct {
	Lines        []RealizationLine
	ProceedsFiat Amount
	CostFiat     Amount
	FeesFiat     Amount
	GainFiat     Amount
}

type Summary struct {
	DisposalProceedsFiatTotal   Amount
	DisposalCostFiatTotal       Amount
	DisposalFeesFiatTotal       Amount
	DisposalGainFiatTotal       Amount
	IncomeFiatTotal             Amount
	DeductibleExpensesFiatTotal Amount
	TaxBaseFiat                 Amount
	TaxDueFiat                  Amount
}

