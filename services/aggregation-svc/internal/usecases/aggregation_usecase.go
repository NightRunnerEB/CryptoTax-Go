package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	pricev1 "github.com/NightRunner/CryptoTax-Go/gen/price/v1"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultImportSource = "ledger"
	defaultPageLimit    = 100
)

type aggregationUC struct {
	txRepo        domain.AggregatedTransactionRepo
	importRepo    domain.ImportStateRepo
	settingsRepo  domain.TenantSettingsRepo
	ledgerClient  domain.LedgerClient
	priceClient   domain.PriceClient
	lockManager   domain.LockManager
	batchSize     int
	importLockTTL time.Duration
}

func NewAggregationUC(
	txRepo domain.AggregatedTransactionRepo,
	importRepo domain.ImportStateRepo,
	settingsRepo domain.TenantSettingsRepo,
	ledgerClient domain.LedgerClient,
	priceClient domain.PriceClient,
	lockManager domain.LockManager,
	batchSize int,
	importLockTTL time.Duration,
) domain.AggregationUseCase {
	return &aggregationUC{
		txRepo:        txRepo,
		importRepo:    importRepo,
		settingsRepo:  settingsRepo,
		ledgerClient:  ledgerClient,
		priceClient:   priceClient,
		lockManager:   lockManager,
		batchSize:     batchSize,
		importLockTTL: importLockTTL,
	}
}

func (u *aggregationUC) ProcessImport(ctx context.Context, event domain.ImportEvent) (retErr error) {
	if err := u.validateDeps(); err != nil {
		return err
	}
	if event.TenantID == uuid.Nil || event.ImportID == uuid.Nil {
		return apperr.InvalidArgument(
			"invalid import event",
			nil,
			apperr.FieldViolation{
				Field:       "event",
				Description: "tenant_id and import_id are required",
			},
		)
	}

	locked, err := u.lockManager.AcquireImportLock(ctx, event.TenantID, event.ImportID, u.importLockTTL)
	if err != nil {
		return apperr.Internal("acquire import lock failed", err, map[string]string{
			"tenant_id": event.TenantID.String(),
			"import_id": event.ImportID.String(),
		})
	}
	if !locked {
		// Another worker is already processing this import.
		return nil
	}
	defer func() {
		if releaseErr := u.lockManager.ReleaseImportLock(ctx, event.TenantID, event.ImportID); releaseErr != nil && retErr == nil {
			retErr = apperr.Internal("release import lock failed", releaseErr, map[string]string{
				"tenant_id": event.TenantID.String(),
				"import_id": event.ImportID.String(),
			})
		}
	}()

	settings, err := u.loadTenantSettings(ctx, event.TenantID)
	if err != nil {
		return err
	}

	state, err := u.importRepo.Get(ctx, event.TenantID, event.ImportID)
	if err != nil && !isNotFound(err) {
		return err
	}
	if err == nil && state.Status == domain.ImportStatusCompleted {
		return nil
	}

	if err := u.importRepo.UpsertProcessing(ctx, domain.AggregationImportState{
		TenantID: event.TenantID,
		ImportID: event.ImportID,
		EventId:  event.EventId,
		Status:   domain.ImportStatusProcessing,
	}); err != nil {
		return err
	}

	ledgerTxs, err := u.ledgerClient.ListTransactionsByImport(ctx, event.TenantID, event.ImportID)
	if err != nil {
		return u.markImportFailed(ctx, event, err)
	}
	if len(ledgerTxs) == 0 {
		if err := u.importRepo.MarkCompleted(ctx, event.TenantID, event.ImportID); err != nil {
			return err
		}
		return nil
	}

	source := pickSource(ledgerTxs)
	priceByTxID, err := u.valuateTransactions(ctx, event.TenantID, source, settings.FiatCurrency, ledgerTxs)
	if err != nil {
		return u.markImportFailed(ctx, event, err)
	}

	aggregated := make([]domain.AggregatedTransaction, 0, len(ledgerTxs))
	for _, tx := range ledgerTxs {
		txID := tx.ID.String()
		valuated := priceByTxID[txID]

		aggregatedTx := domain.AggregatedTransaction{
			ID:             tx.ID,
			TenantID:       event.TenantID,
			Source:         tx.Source,
			ImportID:       event.ImportID,
			TimeUTC:        tx.TimeUTC,
			Kind:           tx.Kind,
			InMoney:        toAggregatedLeg(tx.InMoney, valuedInFiat(valuated)),
			OutMoney:       toAggregatedLeg(tx.OutMoney, valuedOutFiat(valuated)),
			FeeMoney:       toAggregatedLeg(tx.FeeMoney, valuedFeeFiat(valuated)),
			ContractSymbol: tx.ContractSymbol,
			DerivativeKind: tx.DerivativeKind,
			PositionID:     tx.PositionID,
			OrderID:        tx.OrderID,
			TxHash:         tx.TxHash,
			Note:           tx.Note,
			TxFingerprint:  tx.TxFingerprint,
			CreatedAt:      tx.CreatedAt,
		}
		if strings.TrimSpace(aggregatedTx.Source) == "" {
			aggregatedTx.Source = source
		}
		if aggregatedTx.CreatedAt.IsZero() {
			aggregatedTx.CreatedAt = time.Now().UTC()
		}

		aggregated = append(aggregated, aggregatedTx)
	}

	if err := u.txRepo.UpsertBatch(ctx, aggregated); err != nil {
		return u.markImportFailed(ctx, event, err)
	}

	if err := u.importRepo.MarkCompleted(ctx, event.TenantID, event.ImportID); err != nil {
		return err
	}

	return nil
}

func (u *aggregationUC) ListTransactionsByImport(ctx context.Context, tenantID, importID uuid.UUID, limit, offset int32) (domain.AggregatedTxPage, error) {
	if u.txRepo == nil {
		return domain.AggregatedTxPage{}, apperr.Internal("aggregated transaction repo is not configured", nil, nil)
	}
	if tenantID == uuid.Nil || importID == uuid.Nil {
		return domain.AggregatedTxPage{}, apperr.InvalidArgument(
			"invalid request",
			nil,
			apperr.FieldViolation{
				Field:       "tenant_id/import_id",
				Description: "required",
			},
		)
	}
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if offset < 0 {
		return domain.AggregatedTxPage{}, apperr.InvalidArgument(
			"invalid offset",
			nil,
			apperr.FieldViolation{
				Field:       "offset",
				Description: "must be non-negative",
			},
		)
	}

	return u.txRepo.ListByImport(ctx, tenantID, importID, limit, offset)
}

func (u *aggregationUC) ListTransactionsByRange(ctx context.Context, tenantID uuid.UUID, fromUTC, toUTC time.Time, limit, offset int32) (domain.AggregatedTxPage, error) {
	if u.txRepo == nil {
		return domain.AggregatedTxPage{}, apperr.Internal("aggregated transaction repo is not configured", nil, nil)
	}
	if tenantID == uuid.Nil {
		return domain.AggregatedTxPage{}, apperr.InvalidArgument(
			"invalid tenant id",
			nil,
			apperr.FieldViolation{
				Field:       "tenant_id",
				Description: "required",
			},
		)
	}
	if !fromUTC.Before(toUTC) {
		return domain.AggregatedTxPage{}, apperr.InvalidArgument(
			"invalid range",
			nil,
			apperr.FieldViolation{
				Field:       "from_utc/to_utc",
				Description: "from_utc must be before to_utc",
			},
		)
	}
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if offset < 0 {
		return domain.AggregatedTxPage{}, apperr.InvalidArgument(
			"invalid offset",
			nil,
			apperr.FieldViolation{
				Field:       "offset",
				Description: "must be non-negative",
			},
		)
	}

	return u.txRepo.ListByRange(ctx, tenantID, fromUTC, toUTC, limit, offset)
}

func (u *aggregationUC) validateDeps() error {
	switch {
	case u.txRepo == nil:
		return apperr.Internal("aggregated transaction repo is not configured", nil, nil)
	case u.importRepo == nil:
		return apperr.Internal("import state repo is not configured", nil, nil)
	case u.settingsRepo == nil:
		return apperr.Internal("tenant settings repo is not configured", nil, nil)
	case u.ledgerClient == nil:
		return apperr.Internal("ledger client is not configured", nil, nil)
	case u.priceClient == nil:
		return apperr.Internal("price client is not configured", nil, nil)
	case u.lockManager == nil:
		return apperr.Internal("lock manager is not configured", nil, nil)
	default:
		return nil
	}
}

func (u *aggregationUC) loadTenantSettings(ctx context.Context, tenantID uuid.UUID) (domain.TenantSettings, error) {
	settings, err := u.settingsRepo.Get(ctx, tenantID)
	if err != nil {
		if isNotFound(err) {
			return domain.TenantSettings{
				TenantID:     tenantID,
				FiatCurrency: DefaultFiatCurrency,
				Timezone:     DefaultTimezone,
			}, nil
		}
		return domain.TenantSettings{}, err
	}

	settings.FiatCurrency = strings.ToLower(strings.TrimSpace(settings.FiatCurrency))
	if settings.FiatCurrency == "" {
		settings.FiatCurrency = DefaultFiatCurrency
	}
	settings.Timezone = strings.TrimSpace(settings.Timezone)
	if settings.Timezone == "" {
		settings.Timezone = DefaultTimezone
	}

	return settings, nil
}

func (u *aggregationUC) valuateTransactions(
	ctx context.Context,
	tenantID uuid.UUID,
	source string,
	fiatCurrency string,
	ledgerTxs []domain.LedgerTransaction,
) (map[string]*pricev1.ValuatedTx, error) {
	priceByTxID := make(map[string]*pricev1.ValuatedTx, len(ledgerTxs))
	toValuate := make([]*pricev1.TxToValuate, 0, len(ledgerTxs))
	for _, tx := range ledgerTxs {
		toValuate = append(toValuate, &pricev1.TxToValuate{
			TxId:     tx.ID.String(),
			TimeUtc:  timestamppb.New(tx.TimeUTC),
			InMoney:  toPriceMoneyLeg(tx.InMoney),
			OutMoney: toPriceMoneyLeg(tx.OutMoney),
			FeeMoney: toPriceMoneyLeg(tx.FeeMoney),
		})
	}

	batchSize := u.batchSize
	if batchSize <= 0 {
		batchSize = 10000
	}

	for start := 0; start < len(toValuate); start += batchSize {
		end := start + batchSize
		if end > len(toValuate) {
			end = len(toValuate)
		}

		resp, err := u.priceClient.ValuateTransactionsBatch(ctx, &pricev1.ValuateTransactionsRequest{
			TenantId:     tenantID.String(),
			Source:       source,
			FiatCurrency: fiatCurrency,
			Transactions: toValuate[start:end],
		})
		if err != nil {
			return nil, err
		}
		for _, tx := range resp.GetTransactions() {
			if tx == nil {
				continue
			}
			priceByTxID[tx.GetTxId()] = tx
		}
	}

	return priceByTxID, nil
}

func (u *aggregationUC) markImportFailed(ctx context.Context, event domain.ImportEvent, processingErr error) error {
	if processingErr == nil {
		return nil
	}

	message := processingErr.Error()
	if markErr := u.importRepo.MarkFailed(ctx, event.TenantID, event.ImportID, message); markErr != nil {
		return errors.Join(processingErr, markErr)
	}
	return processingErr
}

func toPriceMoneyLeg(asset *domain.LedgerAsset) *pricev1.MoneyLeg {
	if asset == nil {
		return nil
	}
	return &pricev1.MoneyLeg{
		Symbol: strings.TrimSpace(asset.Symbol),
		Amount: strings.TrimSpace(asset.Amount),
	}
}

func toAggregatedLeg(asset *domain.LedgerAsset, fiat *pricev1.FiatLeg) *domain.MoneyLeg {
	if asset == nil {
		return nil
	}

	out := &domain.MoneyLeg{
		Symbol:       strings.TrimSpace(asset.Symbol),
		CryptoAmount: strings.TrimSpace(asset.Amount),
	}

	if fiat == nil {
		return out
	}

	if fiatAmount := strings.TrimSpace(fiat.GetFiat()); fiatAmount != "" {
		out.FiatAmount = &fiatAmount
	}

	if fiatErr := fiat.GetError(); fiatErr != nil {
		mapped := &domain.FiatLegError{
			Code: fiatErr.GetCode().String(),
		}
		if len(fiatErr.GetCandidates()) > 0 {
			mapped.Candidates = make([]domain.FiatLegCandidate, 0, len(fiatErr.GetCandidates()))
			for _, candidate := range fiatErr.GetCandidates() {
				if candidate == nil {
					continue
				}
				mapped.Candidates = append(mapped.Candidates, domain.FiatLegCandidate{
					CoinID: candidate.GetCoinId(),
					Name:   candidate.GetName(),
				})
			}
		}
		out.Error = mapped
	}

	return out
}

func pickSource(txs []domain.LedgerTransaction) string {
	for _, tx := range txs {
		if source := strings.TrimSpace(tx.Source); source != "" {
			return source
		}
	}
	return defaultImportSource
}

func valuedInFiat(tx *pricev1.ValuatedTx) *pricev1.FiatLeg {
	if tx == nil {
		return nil
	}
	return tx.GetInFiat()
}

func valuedOutFiat(tx *pricev1.ValuatedTx) *pricev1.FiatLeg {
	if tx == nil {
		return nil
	}
	return tx.GetOutFiat()
}

func valuedFeeFiat(tx *pricev1.ValuatedTx) *pricev1.FiatLeg {
	if tx == nil {
		return nil
	}
	return tx.GetFeeFiat()
}
