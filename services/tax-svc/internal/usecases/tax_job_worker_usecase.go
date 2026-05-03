package usecases

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/events"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/engines"
)

const (
	ndflKBK = "18210102010011000110"
)

type TaxJobWorkerUC struct {
	jobRepo     domain.TaxJobRepo
	profileRepo domain.TaxProfileRepo
	txProvider  domain.AggregatedTxProvider
	report      domain.ReportClient
	storage     domain.ObjectStorage
	engines     *engines.Registry
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
}

func NewTaxJobWorkerUC(
	jobRepo domain.TaxJobRepo,
	profileRepo domain.TaxProfileRepo,
	txProvider domain.AggregatedTxProvider,
	report domain.ReportClient,
	storage domain.ObjectStorage,
	engineRegistry *engines.Registry,
	maxAttempts int,
	baseDelay time.Duration,
	maxDelay time.Duration,
) *TaxJobWorkerUC {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if baseDelay <= 0 {
		baseDelay = 10 * time.Second
	}
	if maxDelay <= 0 {
		maxDelay = 2 * time.Minute
	}
	if maxDelay < baseDelay {
		maxDelay = baseDelay
	}
	return &TaxJobWorkerUC{
		jobRepo:     jobRepo,
		profileRepo: profileRepo,
		txProvider:  txProvider,
		report:      report,
		storage:     storage,
		engines:     engineRegistry,
		maxAttempts: maxAttempts,
		baseDelay:   baseDelay,
		maxDelay:    maxDelay,
	}
}

func (uc *TaxJobWorkerUC) ProcessNextQueuedJob(ctx context.Context) (bool, error) {
	job, err := uc.jobRepo.ClaimNextQueued(ctx)
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}

	if err := uc.processJob(ctx, *job); err != nil {
		errCode, errMsg := errorForJob(err)
		if uc.shouldRetry(err, job.Attempts) {
			retryAt := time.Now().Add(uc.nextRetryDelay(job.Attempts))
			if requeueErr := uc.jobRepo.Requeue(ctx, job.ID, retryAt, errCode, errMsg); requeueErr != nil {
				return true, requeueErr
			}
			return true, nil
		}
		if markErr := uc.jobRepo.MarkFailed(ctx, job.ID, errCode, errMsg); markErr != nil {
			return true, markErr
		}
	}

	return true, nil
}

func (uc *TaxJobWorkerUC) processJob(ctx context.Context, job domain.TaxJob) error {
	profile, err := uc.profileRepo.Get(ctx, job.UserID)
	if err != nil {
		return err
	}

	if uc.engines == nil {
		return apperr.Internal("engines registry is not configured", nil, nil)
	}
	engine, ok := uc.engines.Resolve(job.PolicySnapshot.Jurisdiction)
	if !ok {
		return apperr.NotImplemented("tax engine for jurisdiction is not implemented", nil, map[string]string{
			"jurisdiction": string(job.PolicySnapshot.Jurisdiction),
		})
	}
	engineName := string(engine.Jurisdiction())

	fromUTC, toUTC, err := taxYearBoundsUTC(job.TaxYear, profile.Timezone)
	if err != nil {
		return apperr.InvalidArgument("invalid profile timezone", err, apperr.FieldViolation{
			Field:       "tax_profile.timezone",
			Description: "must be valid IANA timezone",
		})
	}
	targetFiat := job.PolicySnapshot.Jurisdiction.FiatCurrency()
	if targetFiat == "" {
		return apperr.InvalidArgument("invalid policy jurisdiction", nil, apperr.FieldViolation{
			Field:       "policy_snapshot.jurisdiction",
			Description: "unsupported or missing fiat currency",
		})
	}

	transactions, err := uc.txProvider.ListTransactionsByRange(ctx, job.UserID, fromUTC, toUTC, targetFiat)
	if err != nil {
		var appErr *apperr.Error
		if errors.As(err, &appErr) && appErr != nil {
			return err
		}
		return apperr.AggregationFetchFailed("fetch aggregated transactions failed", err, map[string]string{
			"user_id":  job.UserID.String(),
			"tax_year": fmt.Sprintf("%d", job.TaxYear),
		})
	}

	buildResult, err := engine.Build(ctx, job.UserID, job.PolicySnapshot, transactions)
	if err != nil {
		return err
	}
	summary := summarizeResult(job, profile, buildResult)

	objectKey := fmt.Sprintf("audits/%s/%s.json", job.UserID.String(), job.ID.String())
	auditPayload := map[string]any{
		"report_id":             job.ID.String(),
		"user_id":               job.UserID.String(),
		"tax_year":              job.TaxYear,
		"policy_snapshot":       job.PolicySnapshot,
		"profile":               profile,
		"engine_jurisdiction":   engineName,
		"transactions_count":    len(transactions),
		"engine_version":        "mvp-scaffold",
		"classification_result": buildResult,
		"summary":               summary,
		"warnings":              buildResult.Warnings,
	}
	if err := uc.storage.UploadJSON(ctx, objectKey, auditPayload); err != nil {
		return apperr.MinIOUploadFailed("upload audit artifact failed", err, map[string]string{
			"object_key": objectKey,
			"report_id":  job.ID.String(),
		})
	}

	var reportObjectKey *string
	if job.PolicySnapshot.Jurisdiction == domain.JurisdictionRU {
		ndflPayload, err := buildNDFLPayload(job, profile, summary)
		if err != nil {
			return err
		}
		renderReq := domain.ReportRenderRequest{
			ReportID:     job.ID,
			UserID:       job.UserID,
			Jurisdiction: string(job.PolicySnapshot.Jurisdiction),
			NDFL:         ndflPayload,
		}
		key, err := uc.report.RequestRender(ctx, renderReq)
		if err != nil {
			return apperr.Internal("request report render failed", err, map[string]string{
				"user_id":   job.UserID.String(),
				"report_id": job.ID.String(),
			})
		}
		if strings.TrimSpace(key) != "" {
			reportObjectKey = &key
		}
	}

	return uc.jobRepo.SaveResult(ctx, job.ID, summary, &objectKey, reportObjectKey)
}

func taxYearBoundsUTC(year int, timezone string) (time.Time, time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	fromLocal := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
	toLocal := fromLocal.AddDate(1, 0, 0)
	return fromLocal.UTC(), toLocal.UTC(), nil
}

func errorForJob(err error) (string, string) {
	var ae *apperr.Error
	if asErr := errors.As(err, &ae); asErr && ae != nil {
		return string(ae.Code), ae.Error()
	}
	return string(apperr.ErrInternal), err.Error()
}

func summarizeResult(job domain.TaxJob, profile domain.TaxProfile, result engines.BuildResult) domain.TaxSummary {
	totalIncome := decimal.Zero
	totalTrade := decimal.Zero
	totalExpense := decimal.Zero
	totalP2P := make([]domain.P2PIncome, 0)

	for _, item := range result.RealizationEvents {
		totalIncome = totalIncome.Add(item.ProceedsFiat)
		totalExpense = totalExpense.Add(item.CostBasisFiat)
		if item.Kind == events.RealizationP2PSell {
			totalP2P = append(totalP2P, domain.P2PIncome{
				OccurredAt: item.OccurredAt,
				Qty:        item.Qty,
				GainFiat:   item.ProceedsFiat.Sub(item.CostBasisFiat),
			})
			continue
		}
		totalTrade = totalTrade.Add(item.ProceedsFiat)
	}
	for _, item := range result.IncomeEvents {
		totalIncome = totalIncome.Add(item.AmountFiat)
	}
	for _, item := range result.ExpenseEvents {
		totalExpense = totalExpense.Add(item.AmountFiat)
	}

	taxBase := totalIncome.Sub(totalExpense)
	taxDue := calculateTaxDue(job.PolicySnapshot.Jurisdiction, profile, taxBase)

	return domain.TaxSummary{
		UserID:       job.UserID,
		TaxYear:      job.TaxYear,
		TotalIncome:  totalIncome,
		TotalTrade:   totalTrade,
		TotalP2P:     totalP2P,
		TotalExpense: totalExpense,
		TaxBase:      taxBase,
		TaxDue:       taxDue,
	}
}

func buildNDFLPayload(job domain.TaxJob, profile domain.TaxProfile, summary domain.TaxSummary) (domain.NDFLReportPayload, error) {
	taxToPay := decimal.Zero
	if summary.TaxDue.GreaterThan(decimal.Zero) {
		taxToPay = summary.TaxDue
	}

	inn, err := normalizeINN(profile.INN)
	if err != nil {
		return domain.NDFLReportPayload{}, err
	}

	oktmo, err := normalizeOKTMO(profile.OKTMO)
	if err != nil {
		return domain.NDFLReportPayload{}, err
	}

	taxOfficeCode := deriveTaxOfficeCode(inn)

	reportDate := time.Date(job.TaxYear, time.December, 31, 0, 0, 0, 0, time.UTC)

	appendix2 := make([]domain.NDFLAppendix2Line, 0, len(summary.TotalP2P)+1)
	if summary.TotalTrade.GreaterThan(decimal.Zero) {
		appendix2 = append(appendix2, domain.NDFLAppendix2Line{
			SourceCountryCode:  "999",
			PaymentCountryCode: "643",
			SourceName:         "CRYPTO",
			CurrencyCode:       "643",
			IncomeTypeCode:     "1530",
			IncomeDate:         reportDate,
			FXRate:             decimal.NewFromInt(1),
			IncomeForeign:      summary.TotalTrade,
			IncomeRub:          summary.TotalTrade,
		})
	}
	for _, line := range summary.TotalP2P {
		if line.GainFiat.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if line.OccurredAt.IsZero() {
			return domain.NDFLReportPayload{}, apperr.Internal("p2p income occurred_at is required for ndfl payload", nil, map[string]string{
				"user_id":  job.UserID.String(),
				"tax_year": fmt.Sprintf("%d", job.TaxYear),
			})
		}
		appendix2 = append(appendix2, domain.NDFLAppendix2Line{
			SourceCountryCode:  "999",
			PaymentCountryCode: "643",
			SourceName:         "CRYPTO P2P",
			CurrencyCode:       "643",
			IncomeTypeCode:     "1530",
			IncomeDate:         line.OccurredAt.UTC(),
			FXRate:             decimal.NewFromInt(1),
			IncomeForeign:      line.GainFiat,
			IncomeRub:          line.GainFiat,
		})
	}

	return domain.NDFLReportPayload{
		Header: domain.NDFLHeader{
			TaxYear:          job.TaxYear,
			INN:              inn,
			LastName:         profile.LastName,
			FirstName:        profile.FirstName,
			MiddleName:       profile.MiddleName,
			Phone:            profile.Phone,
			OKTMO:            oktmo,
			TaxResidency:     string(profile.TaxResidencyStatus),
			TaxPayerType:     string(profile.TaxPayerType),
			CorrectionNumber: "0",
			TaxPeriodCode:    "34",
			TaxOfficeCode:    taxOfficeCode,
		},
		Section1: domain.NDFLSection1{
			KBK:         ndflKBK,
			OKTMO:       oktmo,
			TaxToPay:    taxToPay,
			TaxToRefund: decimal.Zero,
		},
		Section2: domain.NDFLSection2{
			IncomeGroupCode: "13",

			TotalIncome:        summary.TotalIncome,
			NonTaxableIncome:   decimal.Zero,
			TaxableIncome:      summary.TotalIncome,
			Deductions:         decimal.Zero,
			RecognizedExpenses: summary.TotalExpense,
			TaxBase:            summary.TaxBase,

			CalculatedTax:      summary.TaxDue,
			WithheldAtSource:   decimal.Zero,
			MaterialBenefitTax: decimal.Zero,
			TradingFeeCredit:   decimal.Zero,
			FixedAdvanceCredit: decimal.Zero,
			ForeignTaxCredit:   decimal.Zero,
			PatentTaxCredit:    decimal.Zero,

			TaxToPay:    taxToPay,
			TaxToRefund: decimal.Zero,

			SimplifiedDeductionRefund: decimal.Zero,
		},
		Appendix2: appendix2,
		Appendix6: domain.NDFLAppendix6{
			OtherPropertyDeduction:      summary.TotalExpense,
			OtherPropertyAcquisitionExp: summary.TotalExpense,
			TotalPropertyDeduction:      summary.TotalExpense,
		},
	}, nil
}

func normalizeINN(inn string) (string, error) {
	value := strings.TrimSpace(inn)
	if len(value) != 12 || !isDigits(value) || !isValidIndividualINN(value) {
		return "", apperr.Internal("invalid INN for ndfl payload", nil, map[string]string{
			"field": "tax_profile.inn",
		})
	}
	return value, nil
}

func normalizeOKTMO(oktmo string) (string, error) {
	value := strings.TrimSpace(oktmo)
	if (len(value) == 8 || len(value) == 11) && isDigits(value) {
		return value, nil
	}
	return "", apperr.Internal("invalid OKTMO for ndfl payload", nil, map[string]string{
		"field": "tax_profile.oktmo",
	})
}

func deriveTaxOfficeCode(inn string) string {
	return inn[:4]
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (uc *TaxJobWorkerUC) shouldRetry(err error, attempts int) bool {
	if attempts >= uc.maxAttempts {
		return false
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae == nil {
		return false
	}

	switch ae.Code {
	case apperr.ErrAggregationUnavailable, apperr.ErrAggregationFetchFailed:
		return shouldRetryByGRPCCode(ae.Cause, ae.Code == apperr.ErrAggregationUnavailable)
	case apperr.ErrStorageUnavailable, apperr.ErrMinIOUploadFailed:
		return true
	case apperr.ErrInternal:
		return shouldRetryByGRPCCode(ae.Cause, false)
	default:
		return false
	}
}

func (uc *TaxJobWorkerUC) nextRetryDelay(attempts int) time.Duration {
	ceiling := uc.backoffCeiling(attempts)
	return fullJitter(ceiling)
}

func (uc *TaxJobWorkerUC) backoffCeiling(attempts int) time.Duration {
	retryNumber := attempts - 1
	if retryNumber < 0 {
		retryNumber = 0
	}

	ceiling := uc.baseDelay
	for i := 0; i < retryNumber; i++ {
		if ceiling >= uc.maxDelay/2 {
			return uc.maxDelay
		}
		ceiling *= 2
	}
	if ceiling > uc.maxDelay {
		return uc.maxDelay
	}
	return ceiling
}

func fullJitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	n, err := crand.Int(crand.Reader, big.NewInt(max.Nanoseconds()+1))
	if err != nil {
		return max / 2
	}
	return time.Duration(n.Int64())
}

func shouldRetryByGRPCCode(err error, fallback bool) bool {
	code, ok := grpcCodeFromErrorChain(err)
	if !ok {
		return fallback
	}
	switch code {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	case codes.InvalidArgument, codes.NotFound, codes.FailedPrecondition, codes.PermissionDenied, codes.Unauthenticated:
		return false
	default:
		return false
	}
}

func grpcCodeFromErrorChain(err error) (codes.Code, bool) {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if st, ok := status.FromError(current); ok {
			return st.Code(), true
		}
	}
	return codes.OK, false
}

func calculateTaxDue(jurisdiction domain.Jurisdiction, profile domain.TaxProfile, taxBase decimal.Decimal) decimal.Decimal {
	if !taxBase.GreaterThan(decimal.Zero) {
		return decimal.Zero
	}

	switch jurisdiction {
	case domain.JurisdictionRU:
		rate := decimal.NewFromFloat(0.13)
		if profile.TaxResidencyStatus == domain.NonResident {
			rate = decimal.NewFromFloat(0.30)
		}
		return taxBase.Mul(rate)
	default:
		return decimal.Zero
	}
}

var _ domain.TaxJobWorkerUseCase = (*TaxJobWorkerUC)(nil)
