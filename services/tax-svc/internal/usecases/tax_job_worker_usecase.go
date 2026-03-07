package usecases

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/engines"
)

type TaxJobWorkerUC struct {
	jobRepo     domain.TaxJobRepo
	profileRepo domain.TaxProfileRepo
	txProvider  domain.AggregatedTxProvider
	report      domain.ReportClient
	storage     domain.ObjectStorage
	engines     *engines.Registry
	presignTTL  time.Duration
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
	presignTTL time.Duration,
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
		presignTTL:  presignTTL,
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
	profile, err := uc.profileRepo.Get(ctx, job.TenantID)
	if err != nil {
		return err
	}

	if uc.engines == nil {
		return apperr.Internal("engines registry is not configured", nil, nil)
	}
	engine, ok := uc.engines.Resolve(profile.Jurisdiction)
	if !ok {
		return apperr.NotImplemented("tax engine for jurisdiction is not implemented", nil, map[string]string{
			"jurisdiction": string(profile.Jurisdiction),
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

	transactions, err := uc.txProvider.ListTransactionsByRange(ctx, job.TenantID, fromUTC, toUTC)
	if err != nil {
		var appErr *apperr.Error
		if errors.As(err, &appErr) && appErr != nil {
			return err
		}
		return apperr.AggregationFetchFailed("fetch aggregated transactions failed", err, map[string]string{
			"tenant_id": job.TenantID.String(),
			"tax_year":  fmt.Sprintf("%d", job.TaxYear),
		})
	}

	buildResult, err := engine.Build(ctx, job.TenantID, job.PolicySnapshot, transactions)
	if err != nil {
		return err
	}
	summary := summarizeResult(job, profile, buildResult)

	objectKey := fmt.Sprintf("audits/%s/%s.json", job.TenantID.String(), job.ID.String())
	auditPayload := map[string]any{
		"report_id":             job.ID.String(),
		"tenant_id":             job.TenantID.String(),
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

	if err := uc.report.RequestRender(ctx, domain.ReportRenderRequest{
		ReportID:         job.ID,
		TenantID:         job.TenantID,
		Jurisdiction:     string(profile.Jurisdiction),
		TaxYear:          int32(job.TaxYear),
		DatasetObjectKey: objectKey,
		TemplateVersion:  "",
	}); err != nil {
		return apperr.Internal("request report render failed", err, map[string]string{
			"tenant_id": job.TenantID.String(),
			"report_id": job.ID.String(),
		})
	}

	auditURL := objectKey
	if url, err := uc.storage.PresignGet(ctx, objectKey, uc.presignTTL); err == nil && url != "" {
		auditURL = url
	}

	return uc.jobRepo.SaveResult(ctx, job.ID, summary, ptr(auditURL), nil)
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

func summarizeResult(job domain.TaxJob, profile domain.TaxProfile, result engines.BuildResult) domain.TaxSummary {
	totalIncome := decimal.Zero
	totalExpense := decimal.Zero

	for _, item := range result.RealizationEvents {
		totalIncome = totalIncome.Add(item.ProceedsFiat)
		totalExpense = totalExpense.Add(item.CostBasisFiat)
	}
	for _, item := range result.IncomeEvents {
		totalIncome = totalIncome.Add(item.AmountFiat)
	}
	for _, item := range result.ExpenseEvents {
		totalExpense = totalExpense.Add(item.AmountFiat)
	}

	taxBase := totalIncome.Sub(totalExpense)
	taxDue := calculateTaxDue(profile, taxBase)

	return domain.TaxSummary{
		TenantID:     job.TenantID,
		TaxYear:      job.TaxYear,
		TotalIncome:  totalIncome,
		TotalExpense: totalExpense,
		TaxBase:      taxBase,
		TaxDue:       taxDue,
	}
}

func calculateTaxDue(profile domain.TaxProfile, taxBase decimal.Decimal) decimal.Decimal {
	if !taxBase.GreaterThan(decimal.Zero) {
		return decimal.Zero
	}

	switch profile.Jurisdiction {
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

func ptr(v string) *string {
	return &v
}

var _ domain.TaxJobWorkerUseCase = (*TaxJobWorkerUC)(nil)
