package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	db "github.com/NightRunner/CryptoTax-Go/services/tax-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/taxcalc"
	"github.com/google/uuid"
)

type reportUC struct {
	store db.Store

	jobRepo      domain.TaxReportJobRepo
	profileRepo  domain.TaxProfileRepo
	taxpayerRepo domain.TaxpayerProfileRepo
	inboxRepo    domain.InboxRepo

	aggregationClient domain.AggregationClient
	storage           domain.ObjectStorage

	defaultJurisdiction string
	defaultTimezone     string
	defaultCostBasis    string

	defaultTreatCryptoToCryptoAsDisposition bool
	templateVersion                         string
	aggregationPageLimit                    int32
	presignTTL                              time.Duration
}

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
	"RUB": {}, "USD": {}, "EUR": {}, "KZT": {}, "GBP": {}, "CHF": {}, "CNY": {}, "JPY": {},
}

type calcAccumulator struct {
	events     []domain.DatasetEvent
	lines      []domain.RealizationLine
	summary    taxcalc.Summary
	targetYear int
	targetLoc  *time.Location
	taxProfile domain.TaxProfile
	engine     *taxcalc.Engine
}

func NewReportUC(
	store db.Store,
	jobRepo domain.TaxReportJobRepo,
	profileRepo domain.TaxProfileRepo,
	taxpayerRepo domain.TaxpayerProfileRepo,
	inboxRepo domain.InboxRepo,
	aggregationClient domain.AggregationClient,
	storage domain.ObjectStorage,
	defaultJurisdiction string,
	defaultTimezone string,
	defaultCostBasis string,
	defaultTreatCryptoToCryptoAsDisposition bool,
	templateVersion string,
	aggregationPageLimit int32,
	presignTTL time.Duration,
) (domain.ReportUseCase, domain.ReportPipelineUseCase) {
	uc := &reportUC{
		store:                                   store,
		jobRepo:                                 jobRepo,
		profileRepo:                             profileRepo,
		taxpayerRepo:                            taxpayerRepo,
		inboxRepo:                               inboxRepo,
		aggregationClient:                       aggregationClient,
		storage:                                 storage,
		defaultJurisdiction:                     strings.TrimSpace(defaultJurisdiction),
		defaultTimezone:                         strings.TrimSpace(defaultTimezone),
		defaultCostBasis:                        strings.TrimSpace(defaultCostBasis),
		defaultTreatCryptoToCryptoAsDisposition: defaultTreatCryptoToCryptoAsDisposition,
		templateVersion:                         strings.TrimSpace(templateVersion),
		aggregationPageLimit:                    aggregationPageLimit,
		presignTTL:                              presignTTL,
	}
	if uc.defaultJurisdiction == "" {
		uc.defaultJurisdiction = "RU"
	}
	if uc.defaultTimezone == "" {
		uc.defaultTimezone = "UTC"
	}
	if uc.defaultCostBasis == "" {
		uc.defaultCostBasis = "FIFO"
	}
	if uc.templateVersion == "" {
		uc.templateVersion = "v1"
	}
	if uc.aggregationPageLimit <= 0 {
		uc.aggregationPageLimit = 1000
	}
	if uc.presignTTL <= 0 {
		uc.presignTTL = 15 * time.Minute
	}

	return uc, uc
}

func (u *reportUC) StartReport(ctx context.Context, params domain.StartReportParams) (domain.TaxReportJob, error) {
	if params.TenantID == uuid.Nil {
		return domain.TaxReportJob{}, apperr.InvalidArgument(
			"invalid tenant id",
			nil,
			apperr.FieldViolation{Field: "tenant_id", Description: "required"},
		)
	}
	if params.TaxYear < 2000 || params.TaxYear > int32(time.Now().UTC().Year()+1) {
		return domain.TaxReportJob{}, apperr.InvalidArgument(
			"invalid tax_year",
			nil,
			apperr.FieldViolation{Field: "tax_year", Description: "must be in valid range"},
		)
	}

	params.Jurisdiction = normalizeOrDefault(params.Jurisdiction, u.defaultJurisdiction)
	params.Timezone = normalizeOrDefault(params.Timezone, u.defaultTimezone)
	params.CostBasisMethod = normalizeOrDefault(params.CostBasisMethod, u.defaultCostBasis)

	rawParams, err := json.Marshal(params)
	if err != nil {
		return domain.TaxReportJob{}, apperr.Internal("marshal report params failed", err, nil)
	}

	reportID := uuid.New()
	jobRequestedEvent := domain.TaxReportJobRequestedEvent{
		EventID:  uuid.New(),
		ReportID: reportID,
		TenantID: params.TenantID,
	}
	payload, err := json.Marshal(jobRequestedEvent)
	if err != nil {
		return domain.TaxReportJob{}, apperr.Internal("marshal outbox payload failed", err, nil)
	}

	if err := u.store.ExecTx(ctx, func(q *db.Queries) error {
		if _, err := q.CreateTaxReportJob(ctx, db.CreateTaxReportJobParams{
			ID:           reportID,
			TenantID:     params.TenantID,
			TaxYear:      params.TaxYear,
			Jurisdiction: params.Jurisdiction,
			Status:       string(domain.ReportJobStatusQueued),
			Params:       rawParams,
		}); err != nil {
			return err
		}
		return q.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
			ID:            uuid.New(),
			AggregateType: "tax_report_job",
			AggregateID:   reportID,
			EventType:     domain.EventTypeTaxReportJobRequested,
			Payload:       payload,
			Status:        string(domain.OutboxStatusPending),
		})
	}); err != nil {
		return domain.TaxReportJob{}, apperr.Internal("start report transaction failed", err, map[string]string{
			"tenant_id": params.TenantID.String(),
			"report_id": reportID.String(),
		})
	}

	return u.jobRepo.Get(ctx, params.TenantID, reportID)
}

func (u *reportUC) GetReportStatus(ctx context.Context, tenantID, reportID uuid.UUID) (domain.ReportStatusView, error) {
	if tenantID == uuid.Nil || reportID == uuid.Nil {
		return domain.ReportStatusView{}, apperr.InvalidArgument(
			"invalid ids",
			nil,
			apperr.FieldViolation{Field: "tenant_id/report_id", Description: "required"},
		)
	}

	job, err := u.jobRepo.Get(ctx, tenantID, reportID)
	if err != nil {
		return domain.ReportStatusView{}, err
	}

	var downloadURL *string
	if job.Status == domain.ReportJobStatusCompleted && job.PDFObjectKey != nil && strings.TrimSpace(*job.PDFObjectKey) != "" {
		if url, presignErr := u.storage.PresignGet(ctx, *job.PDFObjectKey, u.presignTTL); presignErr == nil && strings.TrimSpace(url) != "" {
			downloadURL = &url
		}
	}

	return domain.ReportStatusView{
		Job:         job,
		DownloadURL: downloadURL,
	}, nil
}

func (u *reportUC) ListReports(ctx context.Context, tenantID uuid.UUID, taxYear, limit, offset int32) (domain.TaxReportJobPage, error) {
	if tenantID == uuid.Nil {
		return domain.TaxReportJobPage{}, apperr.InvalidArgument(
			"invalid tenant id",
			nil,
			apperr.FieldViolation{Field: "tenant_id", Description: "required"},
		)
	}
	if limit <= 0 {
		limit = defaultListLimit
	}
	if offset < 0 {
		return domain.TaxReportJobPage{}, apperr.InvalidArgument(
			"invalid offset",
			nil,
			apperr.FieldViolation{Field: "offset", Description: "must be non-negative"},
		)
	}
	return u.jobRepo.List(ctx, tenantID, taxYear, limit, offset)
}

func (u *reportUC) ProcessQueuedReport(ctx context.Context, event domain.TaxReportJobRequestedEvent) error {
	if event.EventID == uuid.Nil || event.ReportID == uuid.Nil || event.TenantID == uuid.Nil {
		return apperr.InvalidArgument("invalid report job event", nil, apperr.FieldViolation{
			Field:       "event",
			Description: "event_id/report_id/tenant_id are required",
		})
	}

	inserted, err := u.inboxRepo.Register(ctx, event.EventID)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	rows, err := u.jobRepo.MarkProcessing(ctx, event.ReportID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return nil
	}

	job, err := u.jobRepo.Get(ctx, event.TenantID, event.ReportID)
	if err != nil {
		return u.failJob(ctx, event.ReportID, wrapJobError("load report job failed", err))
	}

	var params domain.StartReportParams
	if len(job.Params) > 0 {
		if err := json.Unmarshal(job.Params, &params); err != nil {
			return u.failJob(ctx, event.ReportID, apperr.InvalidArgument(
				"invalid report params",
				err,
				apperr.FieldViolation{Field: "params", Description: "invalid json"},
			))
		}
	}

	taxProfile, err := u.loadTaxProfile(ctx, job.TenantID)
	if err != nil {
		return u.failJob(ctx, event.ReportID, wrapJobError("load tax profile failed", err))
	}
	if v := strings.TrimSpace(params.Jurisdiction); v != "" {
		taxProfile.Jurisdiction = v
	}
	if v := strings.TrimSpace(params.Timezone); v != "" {
		taxProfile.Timezone = v
	}
	if v := strings.TrimSpace(params.CostBasisMethod); v != "" {
		taxProfile.CostBasisMethod = v
	}
	if params.TreatCryptoToCryptoAsDisposition {
		taxProfile.TreatSwapAsDisposition = true
	}

	taxpayerProfile, err := u.loadTaxpayerProfile(ctx, job.TenantID)
	if err != nil {
		return u.failJob(ctx, event.ReportID, wrapJobError("load taxpayer profile failed", err))
	}

	transactions, err := u.loadTransactionsForHistory(ctx, job.TenantID, job.TaxYear, taxProfile.Timezone)
	if err != nil {
		return u.failJob(ctx, event.ReportID, wrapJobError("load transactions failed", err))
	}

	acc, err := u.buildReportAccumulator(job, taxProfile, transactions)
	if err != nil {
		return u.failJob(ctx, event.ReportID, err)
	}
	summary := acc.summaryMap()
	dataset := domain.ReportDataset{
		ReportID:         job.ID.String(),
		TenantID:         job.TenantID.String(),
		TaxYear:          job.TaxYear,
		Jurisdiction:     job.Jurisdiction,
		TaxpayerProfile:  taxpayerProfile,
		TaxProfile:       taxProfile,
		Summary:          summary,
		Events:           acc.events,
		RealizationLines: acc.lines,
	}

	datasetObjectKey := fmt.Sprintf("datasets/%s/%s.json", job.TenantID.String(), job.ID.String())
	if err := u.storage.UploadJSON(ctx, datasetObjectKey, dataset); err != nil {
		return u.failJob(ctx, event.ReportID, apperr.MinIOUploadFailed("upload dataset failed", err, map[string]string{
			"report_id": job.ID.String(),
			"tenant_id": job.TenantID.String(),
		}))
	}

	summaryRaw, err := json.Marshal(summary)
	if err != nil {
		return u.failJob(ctx, event.ReportID, apperr.Internal("marshal summary failed", err, map[string]string{
			"report_id": job.ID.String(),
			"tenant_id": job.TenantID.String(),
		}))
	}

	renderEvent := domain.ReportRenderRequestedEvent{
		EventID:          uuid.New(),
		ReportID:         job.ID,
		TenantID:         job.TenantID,
		Jurisdiction:     job.Jurisdiction,
		TaxYear:          job.TaxYear,
		DatasetObjectKey: datasetObjectKey,
		TemplateVersion:  u.templateVersion,
	}
	renderPayload, err := json.Marshal(renderEvent)
	if err != nil {
		return u.failJob(ctx, event.ReportID, apperr.Internal("marshal render event failed", err, map[string]string{
			"report_id": job.ID.String(),
			"tenant_id": job.TenantID.String(),
		}))
	}

	if err := u.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.UpdateTaxReportJobDataset(ctx, db.UpdateTaxReportJobDatasetParams{
			ID:               job.ID,
			DatasetObjectKey: &datasetObjectKey,
			Summary:          summaryRaw,
		}); err != nil {
			return err
		}
		return q.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
			ID:            uuid.New(),
			AggregateType: "tax_report_job",
			AggregateID:   job.ID,
			EventType:     domain.EventTypeReportRenderRequested,
			Payload:       renderPayload,
			Status:        string(domain.OutboxStatusPending),
		})
	}); err != nil {
		return u.failJob(ctx, event.ReportID, apperr.RabbitPublishFailed("persist dataset metadata failed", err, map[string]string{
			"report_id": job.ID.String(),
			"tenant_id": job.TenantID.String(),
		}))
	}

	return nil
}

func (u *reportUC) HandleReportRendered(ctx context.Context, event domain.ReportRenderedEvent) error {
	if event.EventID == uuid.Nil || event.ReportID == uuid.Nil {
		return apperr.InvalidArgument("invalid rendered event", nil, apperr.FieldViolation{
			Field:       "event_id/report_id",
			Description: "required",
		})
	}

	inserted, err := u.inboxRepo.Register(ctx, event.EventID)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	return u.jobRepo.MarkCompleted(ctx, event.ReportID, event.PDFObjectKey)
}

func (u *reportUC) HandleReportRenderFailed(ctx context.Context, event domain.ReportRenderFailedEvent) error {
	if event.EventID == uuid.Nil || event.ReportID == uuid.Nil {
		return apperr.InvalidArgument("invalid render failed event", nil, apperr.FieldViolation{
			Field:       "event_id/report_id",
			Description: "required",
		})
	}

	inserted, err := u.inboxRepo.Register(ctx, event.EventID)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	return u.jobRepo.MarkFailed(ctx, event.ReportID, event.Error)
}

func (u *reportUC) failJob(ctx context.Context, reportID uuid.UUID, processErr error) error {
	errMessage := "UNKNOWN: unknown error"
	if processErr != nil {
		errMessage = formatJobError(processErr)
	}
	if err := u.jobRepo.MarkFailed(ctx, reportID, errMessage); err != nil {
		return err
	}
	return nil
}

func (u *reportUC) loadTaxProfile(ctx context.Context, tenantID uuid.UUID) (domain.TaxProfile, error) {
	profile, err := u.profileRepo.Get(ctx, tenantID)
	if err == nil {
		return profile, nil
	}
	if isNotFound(err) {
		return domain.TaxProfile{
			TenantID:                    tenantID,
			Jurisdiction:                u.defaultJurisdiction,
			CostBasisMethod:             u.defaultCostBasis,
			Timezone:                    u.defaultTimezone,
			TreatSwapAsDisposition:      u.defaultTreatCryptoToCryptoAsDisposition,
			TreatCryptoFeeAsDisposition: defaultTreatCryptoFeeAsDisposition,
			IncludeIncomeEvents:         defaultIncludeIncomeEvents,
			AllowLossEventsDeduction:    defaultAllowLossEventsDeduction,
			FailOnNegativeInventory:     defaultFailOnNegativeInventory,
			FailOnMissingFiat:           defaultFailOnMissingFiat,
		}, nil
	}
	return domain.TaxProfile{}, err
}

func (u *reportUC) loadTaxpayerProfile(ctx context.Context, tenantID uuid.UUID) (domain.TaxpayerProfile, error) {
	profile, err := u.taxpayerRepo.Get(ctx, tenantID)
	if err == nil {
		return profile, nil
	}
	if isNotFound(err) {
		return domain.TaxpayerProfile{TenantID: tenantID}, nil
	}
	return domain.TaxpayerProfile{}, err
}

func (u *reportUC) loadTransactionsForHistory(ctx context.Context, tenantID uuid.UUID, year int32, timezone string) ([]domain.AggregatedTransaction, error) {
	loc, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		return nil, apperr.InvalidArgument("invalid tax profile timezone", err, apperr.FieldViolation{
			Field:       "timezone",
			Description: "unknown timezone",
		})
	}

	from := time.Date(1970, time.January, 1, 0, 0, 0, 0, loc).UTC()
	to := time.Date(int(year)+1, time.January, 1, 0, 0, 0, 0, loc).UTC()
	limit := u.aggregationPageLimit
	offset := int32(0)

	out := make([]domain.AggregatedTransaction, 0, limit)
	for {
		batch, err := u.aggregationClient.ListTransactionsByRange(ctx, tenantID, from, to, limit, offset)
		if err != nil {
			return nil, apperr.AggregationFetchFailed(
				"aggregation fetch failed",
				err,
				map[string]string{
					"tenant_id": tenantID.String(),
					"tax_year":  strconv.FormatInt(int64(year), 10),
				},
			)
		}
		if len(batch) == 0 {
			break
		}

		out = append(out, batch...)
		if len(batch) < int(limit) {
			break
		}
		offset += int32(len(batch))
	}

	return out, nil
}

func (u *reportUC) buildReportAccumulator(job domain.TaxReportJob, profile domain.TaxProfile, txs []domain.AggregatedTransaction) (*calcAccumulator, error) {
	loc, err := time.LoadLocation(strings.TrimSpace(profile.Timezone))
	if err != nil {
		return nil, apperr.InvalidArgument("invalid tax profile timezone", err, apperr.FieldViolation{
			Field:       "timezone",
			Description: "unknown timezone",
		})
	}
	if !strings.EqualFold(strings.TrimSpace(profile.CostBasisMethod), "FIFO") {
		return nil, apperr.InvalidArgument("unsupported cost basis method", nil, apperr.FieldViolation{
			Field:       "cost_basis_method",
			Description: "only FIFO is supported in MVP",
		})
	}

	sortTransactionsDeterministic(txs)

	acc := &calcAccumulator{
		events:     make([]domain.DatasetEvent, 0, len(txs)),
		lines:      make([]domain.RealizationLine, 0, len(txs)),
		targetYear: int(job.TaxYear),
		targetLoc:  loc,
		taxProfile: profile,
		engine: taxcalc.NewEngine(taxcalc.Policy{
			TreatSwapAsDisposition:      profile.TreatSwapAsDisposition,
			TreatCryptoFeeAsDisposition: profile.TreatCryptoFeeAsDisposition,
			IncludeIncomeEvents:         profile.IncludeIncomeEvents,
			AllowLossEventsDeduction:    profile.AllowLossEventsDeduction,
			FailOnNegativeInventory:     profile.FailOnNegativeInventory,
			FailOnMissingFiat:           profile.FailOnMissingFiat,
		}),
	}

	for _, tx := range txs {
		if err := u.applyTransaction(acc, tx); err != nil {
			return nil, err
		}
	}

	acc.summary.TaxBaseFiat = acc.summary.DisposalGainFiatTotal.
		Add(acc.summary.IncomeFiatTotal).
		Sub(acc.summary.DeductibleExpensesFiatTotal)
	if acc.summary.TaxBaseFiat.IsNegative() {
		acc.summary.TaxBaseFiat = taxcalc.Zero()
	}
	acc.summary.TaxDueFiat = calculateRUTaxDue(acc.summary.TaxBaseFiat)

	return acc, nil
}

func (u *reportUC) applyTransaction(acc *calcAccumulator, tx domain.AggregatedTransaction) error {
	inYear := tx.TimeUTC.In(acc.targetLoc).Year() == acc.targetYear
	switch tx.Kind {
	case txKindSpot:
		return u.applySpot(acc, tx, inYear)
	case txKindSwap:
		return u.applySwap(acc, tx, inYear)
	case txKindDepositCrypto, txKindWithdrawalCrypto, txKindTransferInternal:
		acc.addTransferEvent(tx, inYear, firstNonNil(tx.InMoney, tx.OutMoney))
		return nil
	case txKindDepositFiat, txKindWithdrawalFiat:
		acc.addTransferEvent(tx, inYear, firstNonNil(tx.InMoney, tx.OutMoney))
		return nil
	case txKindAirdrop, txKindStakingReward:
		return u.applyIncome(acc, tx, inYear, tx.InMoney)
	case txKindExpense:
		return u.applyExpense(acc, tx, inYear, firstNonNil(tx.OutMoney, tx.FeeMoney))
	case txKindFundingFee:
		return u.applyFundingFee(acc, tx, inYear)
	case txKindDerivativePnL:
		return u.applyDerivativePnL(acc, tx, inYear)
	case txKindGiftIn:
		return u.applyGiftIn(acc, tx, inYear)
	case txKindGiftOut:
		return u.applyGiftOut(acc, tx, inYear)
	case txKindStolen, txKindLost, txKindBurn:
		return u.applyLoss(acc, tx, inYear)
	default:
		return apperr.UnsupportedKind("unsupported transaction kind", nil, map[string]string{
			"tx_id": tx.ID.String(),
			"kind":  tx.Kind,
		})
	}
}

func (u *reportUC) applySpot(acc *calcAccumulator, tx domain.AggregatedTransaction, inYear bool) error {
	if tx.InMoney == nil || tx.OutMoney == nil {
		return invalidTxShape(tx, "Spot transaction requires both in_money and out_money")
	}
	inIsFiat := isFiatSymbol(tx.InMoney.Symbol)
	outIsFiat := isFiatSymbol(tx.OutMoney.Symbol)

	switch {
	case !inIsFiat && outIsFiat:
		qty, err := parsePositiveAmount(tx, "in_money.crypto_amount", tx.InMoney.CryptoAmount)
		if err != nil {
			return err
		}
		costFiat, err := requireFiatAmount(tx, "out_money", tx.OutMoney, acc.taxProfile.FailOnMissingFiat)
		if err != nil {
			return err
		}
		feeFiat, err := extractFiatFee(tx, acc.taxProfile.FailOnMissingFiat)
		if err != nil {
			return err
		}
		totalCost := costFiat.Add(feeFiat)
		if err := acc.engine.ApplyAcquisition(
			tx.InMoney.Symbol,
			qty,
			totalCost,
			tx.TimeUTC,
			tx.ID.String(),
			tx.TxFingerprint,
			"BUY",
			eventMeta(tx),
		); err != nil {
			return err
		}
		if inYear {
			acc.events = append(acc.events, buildEvent(domain.EventAcquisition, tx, tx.InMoney, totalCost, feeFiat))
		}
		return u.applyCryptoFeeDisposition(acc, tx, inYear)

	case inIsFiat && !outIsFiat:
		qty, err := parsePositiveAmount(tx, "out_money.crypto_amount", tx.OutMoney.CryptoAmount)
		if err != nil {
			return err
		}
		proceedsFiat, err := requireFiatAmount(tx, "in_money", tx.InMoney, acc.taxProfile.FailOnMissingFiat)
		if err != nil {
			return err
		}
		feeFiat, err := extractFiatFee(tx, acc.taxProfile.FailOnMissingFiat)
		if err != nil {
			return err
		}

		disposal, err := acc.engine.ApplyDisposition(tx.OutMoney.Symbol, qty, proceedsFiat, feeFiat, tx.ID.String())
		if err != nil {
			return err
		}
		if inYear {
			acc.events = append(acc.events, buildEvent(domain.EventDisposition, tx, tx.OutMoney, proceedsFiat, feeFiat))
			acc.addDisposal(disposal)
		}
		if err := u.applyCryptoFeeDisposition(acc, tx, inYear); err != nil {
			return err
		}
		return nil

	case !inIsFiat && !outIsFiat:
		return u.applySwap(acc, tx, inYear)
	default:
		return invalidTxShape(tx, "unsupported Spot leg shape")
	}
}

func (u *reportUC) applySwap(acc *calcAccumulator, tx domain.AggregatedTransaction, inYear bool) error {
	if tx.InMoney == nil || tx.OutMoney == nil {
		return invalidTxShape(tx, "Swap transaction requires both in_money and out_money")
	}

	outQty, err := parsePositiveAmount(tx, "out_money.crypto_amount", tx.OutMoney.CryptoAmount)
	if err != nil {
		return err
	}
	inQty, err := parsePositiveAmount(tx, "in_money.crypto_amount", tx.InMoney.CryptoAmount)
	if err != nil {
		return err
	}

	if !acc.taxProfile.TreatSwapAsDisposition {
		if err := acc.engine.TransferCostBasis(
			tx.OutMoney.Symbol,
			outQty,
			tx.InMoney.Symbol,
			inQty,
			tx.ID.String(),
			tx.TxFingerprint,
			tx.TimeUTC,
			eventMeta(tx),
		); err != nil {
			return err
		}
		if inYear {
			acc.addTransferEvent(tx, true, tx.InMoney)
			feeFiat, err := extractFiatFee(tx, acc.taxProfile.FailOnMissingFiat)
			if err != nil {
				return err
			}
			if feeFiat.Cmp(taxcalc.Zero()) > 0 {
				acc.events = append(acc.events, buildEvent(domain.EventExpense, tx, tx.FeeMoney, feeFiat, taxcalc.Zero()))
				acc.summary.DeductibleExpensesFiatTotal = acc.summary.DeductibleExpensesFiatTotal.Add(feeFiat)
			}
		}
		return u.applyCryptoFeeDisposition(acc, tx, inYear)
	}

	proceedsFiat, err := requireFiatAmount(tx, "in_money", tx.InMoney, acc.taxProfile.FailOnMissingFiat)
	if err != nil {
		return err
	}
	feeFiat, err := extractFiatFee(tx, acc.taxProfile.FailOnMissingFiat)
	if err != nil {
		return err
	}
	disposal, err := acc.engine.ApplyDisposition(tx.OutMoney.Symbol, outQty, proceedsFiat, feeFiat, tx.ID.String())
	if err != nil {
		return err
	}
	if inYear {
		acc.events = append(acc.events, buildEvent(domain.EventDisposition, tx, tx.OutMoney, proceedsFiat, feeFiat))
		acc.addDisposal(disposal)
	}

	if err := acc.engine.ApplyAcquisition(
		tx.InMoney.Symbol,
		inQty,
		proceedsFiat,
		tx.TimeUTC,
		tx.ID.String(),
		tx.TxFingerprint,
		"SWAP_IN",
		eventMeta(tx),
	); err != nil {
		return err
	}
	if inYear {
		acc.events = append(acc.events, buildEvent(domain.EventAcquisition, tx, tx.InMoney, proceedsFiat, taxcalc.Zero()))
	}
	return u.applyCryptoFeeDisposition(acc, tx, inYear)
}

func (u *reportUC) applyIncome(acc *calcAccumulator, tx domain.AggregatedTransaction, inYear bool, leg *domain.MoneyLeg) error {
	if leg == nil {
		return invalidTxShape(tx, "income transaction requires in_money")
	}
	qty, err := parsePositiveAmount(tx, "in_money.crypto_amount", leg.CryptoAmount)
	if err != nil {
		return err
	}
	incomeFiat, err := requireFiatAmount(tx, "in_money", leg, acc.taxProfile.FailOnMissingFiat)
	if err != nil {
		return err
	}

	if inYear {
		acc.events = append(acc.events, buildEvent(domain.EventIncome, tx, leg, incomeFiat, taxcalc.Zero()))
	}

	if acc.taxProfile.IncludeIncomeEvents && inYear {
		acc.summary.IncomeFiatTotal = acc.summary.IncomeFiatTotal.Add(incomeFiat)
	}
	if acc.taxProfile.IncludeIncomeEvents && !isFiatSymbol(leg.Symbol) {
		if err := acc.engine.ApplyAcquisition(
			leg.Symbol,
			qty,
			incomeFiat,
			tx.TimeUTC,
			tx.ID.String(),
			tx.TxFingerprint,
			"INCOME",
			eventMeta(tx),
		); err != nil {
			return err
		}
	}

	return u.applyCryptoFeeDisposition(acc, tx, inYear)
}

func (u *reportUC) applyExpense(acc *calcAccumulator, tx domain.AggregatedTransaction, inYear bool, leg *domain.MoneyLeg) error {
	if leg == nil {
		return invalidTxShape(tx, "expense transaction requires out_money or fee_money")
	}
	if isFiatSymbol(leg.Symbol) {
		expenseFiat, err := requireFiatAmount(tx, "out_money", leg, acc.taxProfile.FailOnMissingFiat)
		if err != nil {
			return err
		}
		if inYear {
			acc.events = append(acc.events, buildEvent(domain.EventExpense, tx, leg, expenseFiat, taxcalc.Zero()))
			acc.summary.DeductibleExpensesFiatTotal = acc.summary.DeductibleExpensesFiatTotal.Add(expenseFiat)
		}
		if tx.FeeMoney != nil && leg != tx.FeeMoney {
			return u.applyCryptoFeeDisposition(acc, tx, inYear)
		}
		return nil
	}

	qty, err := parsePositiveAmount(tx, "out_money.crypto_amount", leg.CryptoAmount)
	if err != nil {
		return err
	}
	disposal, err := acc.engine.ApplyDisposition(leg.Symbol, qty, taxcalc.Zero(), taxcalc.Zero(), tx.ID.String())
	if err != nil {
		return err
	}
	if inYear {
		acc.events = append(acc.events, buildEvent(domain.EventDisposition, tx, leg, taxcalc.Zero(), taxcalc.Zero()))
		acc.addDisposal(disposal)
	}
	if tx.FeeMoney != nil && leg != tx.FeeMoney {
		return u.applyCryptoFeeDisposition(acc, tx, inYear)
	}
	return nil
}

func (u *reportUC) applyFundingFee(acc *calcAccumulator, tx domain.AggregatedTransaction, inYear bool) error {
	leg := firstNonNil(tx.FeeMoney, tx.OutMoney)
	if leg == nil {
		return invalidTxShape(tx, "FundingFee requires fee_money or out_money")
	}
	return u.applyExpense(acc, tx, inYear, leg)
}

func (u *reportUC) applyDerivativePnL(acc *calcAccumulator, tx domain.AggregatedTransaction, inYear bool) error {
	if tx.InMoney != nil {
		return u.applyIncome(acc, tx, inYear, tx.InMoney)
	}
	if tx.OutMoney != nil {
		return u.applyExpense(acc, tx, inYear, tx.OutMoney)
	}
	return invalidTxShape(tx, "DerivativePnL requires in_money or out_money")
}

func (u *reportUC) applyGiftIn(acc *calcAccumulator, tx domain.AggregatedTransaction, inYear bool) error {
	if tx.InMoney == nil {
		return invalidTxShape(tx, "GiftIn requires in_money")
	}
	qty, err := parsePositiveAmount(tx, "in_money.crypto_amount", tx.InMoney.CryptoAmount)
	if err != nil {
		return err
	}
	if err := acc.engine.ApplyAcquisition(
		tx.InMoney.Symbol,
		qty,
		taxcalc.Zero(),
		tx.TimeUTC,
		tx.ID.String(),
		tx.TxFingerprint,
		"GIFT_IN",
		eventMeta(tx),
	); err != nil {
		return err
	}
	if inYear {
		acc.events = append(acc.events, buildEvent(domain.EventGiftIn, tx, tx.InMoney, taxcalc.Zero(), taxcalc.Zero()))
	}
	return nil
}

func (u *reportUC) applyGiftOut(acc *calcAccumulator, tx domain.AggregatedTransaction, inYear bool) error {
	if tx.OutMoney == nil {
		return invalidTxShape(tx, "GiftOut requires out_money")
	}
	qty, err := parsePositiveAmount(tx, "out_money.crypto_amount", tx.OutMoney.CryptoAmount)
	if err != nil {
		return err
	}
	disposal, err := acc.engine.ApplyDisposition(tx.OutMoney.Symbol, qty, taxcalc.Zero(), taxcalc.Zero(), tx.ID.String())
	if err != nil {
		return err
	}
	if inYear {
		acc.events = append(acc.events, buildEvent(domain.EventGiftOut, tx, tx.OutMoney, taxcalc.Zero(), taxcalc.Zero()))
		acc.addRealizationLines(disposal.Lines)
	}
	return nil
}

func (u *reportUC) applyLoss(acc *calcAccumulator, tx domain.AggregatedTransaction, inYear bool) error {
	leg := firstNonNil(tx.OutMoney, tx.InMoney)
	if leg == nil {
		return invalidTxShape(tx, "Loss transaction requires in_money or out_money")
	}
	qty, err := parsePositiveAmount(tx, "loss.crypto_amount", leg.CryptoAmount)
	if err != nil {
		return err
	}
	disposal, err := acc.engine.ApplyDisposition(leg.Symbol, qty, taxcalc.Zero(), taxcalc.Zero(), tx.ID.String())
	if err != nil {
		return err
	}
	if inYear {
		acc.events = append(acc.events, buildEvent(domain.EventLoss, tx, leg, taxcalc.Zero(), taxcalc.Zero()))
		acc.addRealizationLines(disposal.Lines)
		if acc.taxProfile.AllowLossEventsDeduction {
			acc.summary.DeductibleExpensesFiatTotal = acc.summary.DeductibleExpensesFiatTotal.Add(disposal.CostFiat)
		}
	}
	return nil
}

func (u *reportUC) applyCryptoFeeDisposition(acc *calcAccumulator, tx domain.AggregatedTransaction, inYear bool) error {
	if tx.FeeMoney == nil {
		return nil
	}
	if isFiatSymbol(tx.FeeMoney.Symbol) {
		return nil
	}
	if !acc.taxProfile.TreatCryptoFeeAsDisposition {
		return nil
	}
	qty, err := parsePositiveAmount(tx, "fee_money.crypto_amount", tx.FeeMoney.CryptoAmount)
	if err != nil {
		return err
	}

	disposal, err := acc.engine.ApplyDisposition(tx.FeeMoney.Symbol, qty, taxcalc.Zero(), taxcalc.Zero(), tx.ID.String())
	if err != nil {
		return err
	}
	if inYear {
		acc.events = append(acc.events, buildEvent(domain.EventDisposition, tx, tx.FeeMoney, taxcalc.Zero(), taxcalc.Zero()))
		acc.addDisposal(disposal)
	}
	return nil
}

func sortTransactionsDeterministic(txs []domain.AggregatedTransaction) {
	sort.Slice(txs, func(i, j int) bool {
		left := txs[i]
		right := txs[j]
		if left.TimeUTC.Equal(right.TimeUTC) {
			leftID := left.ID.String()
			rightID := right.ID.String()
			if leftID == rightID {
				return left.TxFingerprint < right.TxFingerprint
			}
			return leftID < rightID
		}
		return left.TimeUTC.Before(right.TimeUTC)
	})
}

func normalizeOrDefault(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func formatJobError(err error) string {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return string(ae.Code) + ": " + ae.Msg
	}
	return "INTERNAL_ERROR: " + err.Error()
}

func wrapJobError(prefix string, err error) error {
	if err == nil {
		return apperr.Internal(prefix, nil, nil)
	}
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return &apperr.Error{
			Op:      ae.Op,
			Code:    ae.Code,
			Msg:     prefix + ": " + ae.Msg,
			Meta:    ae.Meta,
			Details: ae.Details,
			Cause:   ae.Cause,
		}
	}
	return apperr.Internal(prefix, err, nil)
}

func buildEvent(
	eventType domain.DatasetEventType,
	tx domain.AggregatedTransaction,
	leg *domain.MoneyLeg,
	fiat taxcalc.Amount,
	feeFiat taxcalc.Amount,
) domain.DatasetEvent {
	out := domain.DatasetEvent{
		EventType:     eventType,
		TxID:          tx.ID.String(),
		TimeUTC:       tx.TimeUTC,
		Source:        tx.Source,
		Kind:          tx.Kind,
		TxFingerprint: tx.TxFingerprint,
		Meta:          eventMeta(tx),
	}
	if leg != nil {
		out.AssetSymbol = strings.ToUpper(strings.TrimSpace(leg.Symbol))
		out.CryptoAmount = strings.TrimSpace(leg.CryptoAmount)
	}
	if fiat.Cmp(taxcalc.Zero()) != 0 {
		out.FiatAmount = ptrFromAmount(fiat)
	}
	if feeFiat.Cmp(taxcalc.Zero()) > 0 {
		out.FeeFiatAmount = ptrFromAmount(feeFiat)
	}
	return out
}

func (acc *calcAccumulator) addTransferEvent(tx domain.AggregatedTransaction, inYear bool, leg *domain.MoneyLeg) {
	if !inYear {
		return
	}
	acc.events = append(acc.events, buildEvent(domain.EventTransfer, tx, leg, taxcalc.Zero(), taxcalc.Zero()))
}

func (acc *calcAccumulator) addDisposal(disposal taxcalc.DisposalResult) {
	acc.summary.DisposalProceedsFiatTotal = acc.summary.DisposalProceedsFiatTotal.Add(disposal.ProceedsFiat)
	acc.summary.DisposalCostFiatTotal = acc.summary.DisposalCostFiatTotal.Add(disposal.CostFiat)
	acc.summary.DisposalFeesFiatTotal = acc.summary.DisposalFeesFiatTotal.Add(disposal.FeesFiat)
	acc.summary.DisposalGainFiatTotal = acc.summary.DisposalGainFiatTotal.Add(disposal.GainFiat)
	acc.addRealizationLines(disposal.Lines)
}

func (acc *calcAccumulator) addRealizationLines(lines []taxcalc.RealizationLine) {
	if len(lines) == 0 {
		return
	}
	for _, line := range lines {
		acc.lines = append(acc.lines, domain.RealizationLine{
			DisposalTxID:      line.DisposalTxID,
			AssetSymbol:       line.AssetSymbol,
			QtyDisposed:       line.QtyDisposed.String(),
			ProceedsFiatAlloc: line.ProceedsFiatAlloc.String(),
			CostFiatAlloc:     line.CostFiatAlloc.String(),
			FeesFiatAlloc:     line.FeesFiatAlloc.String(),
			GainFiatAlloc:     line.GainFiatAlloc.String(),
			LotAcquiredAt:     line.LotAcquiredAt,
			LotSourceTxID:     line.LotSourceTxID,
		})
	}
}

func (acc *calcAccumulator) summaryMap() map[string]any {
	return map[string]any{
		"disposal_proceeds_fiat_total":   acc.summary.DisposalProceedsFiatTotal.String(),
		"disposal_cost_fiat_total":       acc.summary.DisposalCostFiatTotal.String(),
		"disposal_fees_fiat_total":       acc.summary.DisposalFeesFiatTotal.String(),
		"disposal_gain_fiat_total":       acc.summary.DisposalGainFiatTotal.String(),
		"income_fiat_total":              acc.summary.IncomeFiatTotal.String(),
		"deductible_expenses_fiat_total": acc.summary.DeductibleExpensesFiatTotal.String(),
		"tax_base_fiat":                  acc.summary.TaxBaseFiat.String(),
		"tax_due_fiat":                   acc.summary.TaxDueFiat.String(),
	}
}

func calculateRUTaxDue(taxBase taxcalc.Amount) taxcalc.Amount {
	if taxBase.Cmp(taxcalc.Zero()) <= 0 {
		return taxcalc.Zero()
	}
	threshold := taxcalc.NewInt(2_400_000)
	rateLow := taxcalc.MustParse("0.13")
	rateHigh := taxcalc.MustParse("0.15")
	if taxBase.Cmp(threshold) <= 0 {
		return taxBase.Mul(rateLow)
	}
	lowPart := threshold.Mul(rateLow)
	highPart := taxBase.Sub(threshold).Mul(rateHigh)
	return lowPart.Add(highPart)
}

func extractFiatFee(tx domain.AggregatedTransaction, failOnMissingFiat bool) (taxcalc.Amount, error) {
	if tx.FeeMoney == nil {
		return taxcalc.Zero(), nil
	}
	if !isFiatSymbol(tx.FeeMoney.Symbol) {
		return taxcalc.Zero(), nil
	}
	return requireFiatAmount(tx, "fee_money", tx.FeeMoney, failOnMissingFiat)
}

func requireFiatAmount(tx domain.AggregatedTransaction, field string, leg *domain.MoneyLeg, failOnMissingFiat bool) (taxcalc.Amount, error) {
	if leg == nil {
		return taxcalc.Zero(), invalidTxShape(tx, field+" is required")
	}
	if leg.Error != nil && failOnMissingFiat {
		return taxcalc.Zero(), apperr.NeedsPriceResolution("needs price resolution", nil, map[string]string{
			"tx_id":   tx.ID.String(),
			"kind":    tx.Kind,
			"field":   field,
			"symbol":  strings.ToUpper(strings.TrimSpace(leg.Symbol)),
			"reason":  "price leg contains error",
			"leg_err": strings.TrimSpace(leg.Error.Code),
		})
	}

	if leg.FiatAmount != nil && strings.TrimSpace(*leg.FiatAmount) != "" {
		value, err := taxcalc.Parse(strings.TrimSpace(*leg.FiatAmount))
		if err != nil {
			return taxcalc.Zero(), apperr.InvalidArgument("invalid fiat amount", err, apperr.FieldViolation{
				Field:       field + ".fiat_amount",
				Description: "invalid decimal",
			})
		}
		return value, nil
	}

	if isFiatSymbol(leg.Symbol) {
		value, err := taxcalc.Parse(strings.TrimSpace(leg.CryptoAmount))
		if err != nil {
			return taxcalc.Zero(), apperr.InvalidArgument("invalid fiat amount", err, apperr.FieldViolation{
				Field:       field + ".crypto_amount",
				Description: "invalid decimal",
			})
		}
		return value, nil
	}

	if failOnMissingFiat {
		return taxcalc.Zero(), apperr.NeedsPriceResolution("needs price resolution", nil, map[string]string{
			"tx_id":  tx.ID.String(),
			"kind":   tx.Kind,
			"field":  field,
			"symbol": strings.ToUpper(strings.TrimSpace(leg.Symbol)),
			"reason": "fiat amount is empty",
		})
	}
	return taxcalc.Zero(), nil
}

func parsePositiveAmount(tx domain.AggregatedTransaction, field, raw string) (taxcalc.Amount, error) {
	value, err := taxcalc.Parse(strings.TrimSpace(raw))
	if err != nil {
		return taxcalc.Zero(), apperr.InvalidArgument("invalid decimal amount", err, apperr.FieldViolation{
			Field:       field,
			Description: "invalid decimal",
		})
	}
	if value.Cmp(taxcalc.Zero()) <= 0 {
		return taxcalc.Zero(), invalidTxShape(tx, field+" must be positive")
	}
	return value, nil
}

func isFiatSymbol(symbol string) bool {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	_, ok := fiatSymbols[symbol]
	return ok
}

func firstNonNil(legs ...*domain.MoneyLeg) *domain.MoneyLeg {
	for _, leg := range legs {
		if leg != nil {
			return leg
		}
	}
	return nil
}

func ptrFromAmount(v taxcalc.Amount) *string {
	out := v.String()
	return &out
}

func invalidTxShape(tx domain.AggregatedTransaction, msg string) error {
	return apperr.InvalidTxShape(msg, nil, map[string]string{
		"tx_id": tx.ID.String(),
		"kind":  tx.Kind,
	})
}

func eventMeta(tx domain.AggregatedTransaction) map[string]string {
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

func ptrString(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	out := v
	return &out
}

var _ domain.ReportUseCase = (*reportUC)(nil)
var _ domain.ReportPipelineUseCase = (*reportUC)(nil)
