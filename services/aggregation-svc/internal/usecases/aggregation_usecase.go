package usecase

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pricev1 "github.com/NightRunner/CryptoTax-Go/gen/price/v1"
	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
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
	startedAt := time.Now()
	log := logger.FromContext(ctx).With(
		zap.String("tenant_id", event.TenantID.String()),
		zap.String("import_id", event.ImportID.String()),
		zap.String("event_id", event.EventId.String()),
	)
	log.Debug("ProcessImport: started")
	defer func() {
		if retErr != nil {
			log.Warn("ProcessImport: finished with error",
				zap.Error(retErr),
				zap.Duration("elapsed", time.Since(startedAt)),
			)
			return
		}
		log.Debug("ProcessImport: finished successfully", zap.Duration("elapsed", time.Since(startedAt)))
	}()

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
		log.Debug("ProcessImport: import lock not acquired, skip")
		return nil
	}
	log.Debug("ProcessImport: import lock acquired", zap.Duration("lock_ttl", u.importLockTTL))
	defer func() {
		if releaseErr := u.lockManager.ReleaseImportLock(ctx, event.TenantID, event.ImportID); releaseErr != nil && retErr == nil {
			retErr = apperr.Internal("release import lock failed", releaseErr, map[string]string{
				"tenant_id": event.TenantID.String(),
				"import_id": event.ImportID.String(),
			})
		} else if releaseErr == nil {
			log.Debug("ProcessImport: import lock released")
		}
	}()

	settings, err := u.loadTenantSettings(ctx, event.TenantID)
	if err != nil {
		return err
	}
	log.Debug("ProcessImport: tenant settings loaded",
		zap.String("fiat_currency", settings.FiatCurrency),
		zap.String("timezone", settings.Timezone),
	)

	state, err := u.importRepo.Get(ctx, event.TenantID, event.ImportID)
	if err != nil && !isNotFound(err) {
		return err
	}
	if err == nil && state.Status == domain.ImportStatusCompleted {
		log.Debug("ProcessImport: import already completed, skip")
		return nil
	}
	if err == nil {
		log.Debug("ProcessImport: import state found", zap.String("status", string(state.Status)))
	} else {
		log.Debug("ProcessImport: import state not found, creating processing state")
	}

	if err := u.importRepo.UpsertProcessing(ctx, domain.AggregationImportState{
		TenantID: event.TenantID,
		ImportID: event.ImportID,
		EventId:  event.EventId,
		Status:   domain.ImportStatusProcessing,
	}); err != nil {
		return err
	}
	log.Debug("ProcessImport: import state set to processing")

	ledgerTxs, err := u.ledgerClient.ListTransactionsByImport(ctx, event.TenantID, event.ImportID)
	if err != nil {
		return u.markImportFailed(ctx, event, err)
	}
	log.Debug("ProcessImport: ledger transactions loaded", zap.Int("count", len(ledgerTxs)))
	log.Debug("ProcessImport: ledger tx fingerprints", zap.Any("tx_fingerprints", ledgerTxFingerprints(ledgerTxs)))
	if len(ledgerTxs) == 0 {
		if err := u.importRepo.MarkCompleted(ctx, event.TenantID, event.ImportID); err != nil {
			return err
		}
		log.Debug("ProcessImport: no ledger transactions, marked completed")
		return nil
	}

	source := pickSource(ledgerTxs)
	log.Debug("ProcessImport: start valuation",
		zap.String("source", source),
		zap.String("fiat_currency", settings.FiatCurrency),
		zap.Int("batch_size", u.batchSize),
	)
	priceByTxID, err := u.valuateTransactions(ctx, event.TenantID, source, settings.FiatCurrency, ledgerTxs)
	if err != nil {
		return u.markImportFailed(ctx, event, err)
	}
	log.Debug("ProcessImport: valuation completed", zap.Int("valuated_count", len(priceByTxID)))

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
	log.Debug("ProcessImport: aggregated transactions prepared", zap.Int("count", len(aggregated)))

	if err := u.txRepo.UpsertBatch(ctx, aggregated); err != nil {
		return u.markImportFailed(ctx, event, err)
	}
	log.Debug("ProcessImport: aggregated transactions persisted", zap.Int("count", len(aggregated)))

	if err := u.importRepo.MarkCompleted(ctx, event.TenantID, event.ImportID); err != nil {
		return err
	}
	log.Debug("ProcessImport: import state marked completed")

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

func (u *aggregationUC) ListTransactionsByRange(
	ctx context.Context,
	tenantID uuid.UUID,
	fromUTC, toUTC time.Time,
	limit, offset int32,
	targetFiat string,
) (domain.AggregatedTxPage, error) {
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

	page, err := u.txRepo.ListByRange(ctx, tenantID, fromUTC, toUTC, limit, offset)
	if err != nil {
		return domain.AggregatedTxPage{}, err
	}
	targetFiat = strings.ToUpper(strings.TrimSpace(targetFiat))
	if targetFiat != "" {
		revalued, err := u.revaluateTransactionsToTargetFiat(ctx, tenantID, targetFiat, page.Transactions)
		if err != nil {
			return domain.AggregatedTxPage{}, err
		}
		page.Transactions = revalued
	}
	if err := ensureTransactionsReadyForTax(page.Transactions); err != nil {
		return domain.AggregatedTxPage{}, err
	}
	return page, nil
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
	settings.FiatCurrency = normalizeFiatForPricing(settings.FiatCurrency)

	return settings, nil
}

func (u *aggregationUC) valuateTransactions(
	ctx context.Context,
	tenantID uuid.UUID,
	source string,
	fiatCurrency string,
	ledgerTxs []domain.LedgerTransaction,
) (map[string]*pricev1.ValuatedTx, error) {
	log := logger.FromContext(ctx)
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
		log.Debug("valuateTransactions: sending batch",
			zap.Int("start", start),
			zap.Int("end", end),
			zap.Int("size", end-start),
			zap.String("source", source),
			zap.String("fiat_currency", fiatCurrency),
		)

		resp, err := u.priceClient.ValuateTransactionsBatch(ctx, &pricev1.ValuateTransactionsRequest{
			TenantId:     tenantID.String(),
			Source:       source,
			FiatCurrency: fiatCurrency,
			Transactions: toValuate[start:end],
		})
		if err != nil {
			return nil, mapPriceValuationError("price valuation failed", err, map[string]string{
				"tenant_id":     tenantID.String(),
				"source":        source,
				"fiat_currency": fiatCurrency,
			})
		}
		log.Debug("valuateTransactions: batch response received",
			zap.Int("response_size", len(resp.GetTransactions())),
		)
		for _, tx := range resp.GetTransactions() {
			if tx == nil {
				continue
			}
			priceByTxID[tx.GetTxId()] = tx
		}
	}

	return priceByTxID, nil
}

func (u *aggregationUC) revaluateTransactionsToTargetFiat(
	ctx context.Context,
	tenantID uuid.UUID,
	targetFiat string,
	txs []domain.AggregatedTransaction,
) ([]domain.AggregatedTransaction, error) {
	if len(txs) == 0 {
		return txs, nil
	}

	batchSize := u.batchSize
	if batchSize <= 0 {
		batchSize = 10000
	}

	grouped := make(map[string][]*pricev1.TxToValuate)
	sourcesOrder := make([]string, 0, 4)
	for _, tx := range txs {
		source := strings.TrimSpace(tx.Source)
		if source == "" {
			source = defaultImportSource
		}
		if _, ok := grouped[source]; !ok {
			sourcesOrder = append(sourcesOrder, source)
		}
		grouped[source] = append(grouped[source], &pricev1.TxToValuate{
			TxId:     tx.ID.String(),
			TimeUtc:  timestamppb.New(tx.TimeUTC),
			InMoney:  toPriceMoneyLegFromAggregated(tx.InMoney),
			OutMoney: toPriceMoneyLegFromAggregated(tx.OutMoney),
			FeeMoney: toPriceMoneyLegFromAggregated(tx.FeeMoney),
		})
	}

	valuationsByTxID := make(map[string]*pricev1.ValuatedTx, len(txs))
	for _, source := range sourcesOrder {
		items := grouped[source]
		for start := 0; start < len(items); start += batchSize {
			end := start + batchSize
			if end > len(items) {
				end = len(items)
			}

			resp, err := u.priceClient.ValuateTransactionsBatch(ctx, &pricev1.ValuateTransactionsRequest{
				TenantId:     tenantID.String(),
				Source:       source,
				FiatCurrency: targetFiat,
				Transactions: items[start:end],
			})
			if err != nil {
				return nil, mapPriceValuationError("price revaluation failed", err, map[string]string{
					"tenant_id":   tenantID.String(),
					"source":      source,
					"target_fiat": targetFiat,
				})
			}

			for _, tx := range resp.GetTransactions() {
				if tx == nil {
					continue
				}
				valuationsByTxID[tx.GetTxId()] = tx
			}
		}
	}

	revalued := make([]domain.AggregatedTransaction, 0, len(txs))
	for _, tx := range txs {
		valuated := valuationsByTxID[tx.ID.String()]
		if valuated == nil && (tx.InMoney != nil || tx.OutMoney != nil || tx.FeeMoney != nil) {
			return nil, apperr.DataNotReady(
				"aggregated data is not ready for requested fiat revaluation",
				nil,
				map[string]string{
					"tenant_id":   tenantID.String(),
					"tx_id":       tx.ID.String(),
					"target_fiat": targetFiat,
				},
				apperr.Validation{Violations: []apperr.FieldViolation{
					{
						Field:       "target_fiat",
						Description: "revaluation response is incomplete",
					},
				}},
			)
		}

		item := tx
		item.InMoney = mergeLegFiatValuation(tx.InMoney, valuedInFiat(valuated))
		item.OutMoney = mergeLegFiatValuation(tx.OutMoney, valuedOutFiat(valuated))
		item.FeeMoney = mergeLegFiatValuation(tx.FeeMoney, valuedFeeFiat(valuated))
		revalued = append(revalued, item)
	}

	return revalued, nil
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

func toPriceMoneyLegFromAggregated(leg *domain.MoneyLeg) *pricev1.MoneyLeg {
	if leg == nil {
		return nil
	}
	return &pricev1.MoneyLeg{
		Symbol: strings.TrimSpace(leg.Symbol),
		Amount: strings.TrimSpace(leg.CryptoAmount),
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

func mergeLegFiatValuation(leg *domain.MoneyLeg, fiat *pricev1.FiatLeg) *domain.MoneyLeg {
	if leg == nil {
		return nil
	}

	out := &domain.MoneyLeg{
		Symbol:       strings.TrimSpace(leg.Symbol),
		CryptoAmount: strings.TrimSpace(leg.CryptoAmount),
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

func ledgerTxFingerprints(txs []domain.LedgerTransaction) []map[string]string {
	out := make([]map[string]string, 0, len(txs))
	for _, tx := range txs {
		out = append(out, map[string]string{
			"tx_id":          tx.ID.String(),
			"tx_fingerprint": tx.TxFingerprint,
		})
	}
	return out
}

func ensureTransactionsReadyForTax(txs []domain.AggregatedTransaction) error {
	unresolvedCount := 0
	firstTxID := ""
	firstLeg := ""
	firstReason := ""

	for _, tx := range txs {
		if leg, reason := legNotReadyReason(tx.InMoney); reason != "" {
			unresolvedCount++
			if firstTxID == "" {
				firstTxID = tx.ID.String()
				firstLeg = legName("in_money", leg)
				firstReason = reason
			}
		}
		if leg, reason := legNotReadyReason(tx.OutMoney); reason != "" {
			unresolvedCount++
			if firstTxID == "" {
				firstTxID = tx.ID.String()
				firstLeg = legName("out_money", leg)
				firstReason = reason
			}
		}
		if leg, reason := legNotReadyReason(tx.FeeMoney); reason != "" {
			unresolvedCount++
			if firstTxID == "" {
				firstTxID = tx.ID.String()
				firstLeg = legName("fee_money", leg)
				firstReason = reason
			}
		}
	}

	if unresolvedCount == 0 {
		return nil
	}

	violations := []apperr.FieldViolation{
		{
			Field:       "transactions",
			Description: "contains unresolved fiat valuations",
		},
	}
	if firstLeg != "" {
		violations = append(violations, apperr.FieldViolation{
			Field:       firstLeg,
			Description: firstReason,
		})
	}

	return apperr.DataNotReady(
		"aggregated data is not ready for tax calculation; resolve fiat valuations and retry",
		nil,
		map[string]string{
			"unresolved_count": strconv.Itoa(unresolvedCount),
			"first_tx_id":      firstTxID,
			"first_leg":        firstLeg,
		},
		apperr.Validation{Violations: violations},
	)
}

func legNotReadyReason(leg *domain.MoneyLeg) (string, string) {
	if leg == nil {
		return "", ""
	}
	if leg.Error != nil {
		if code := strings.TrimSpace(leg.Error.Code); code != "" {
			return leg.Symbol, "fiat error: " + code
		}
		return leg.Symbol, "fiat error"
	}
	if leg.FiatAmount == nil || strings.TrimSpace(*leg.FiatAmount) == "" {
		return leg.Symbol, "missing fiat_amount"
	}
	return "", ""
}

func legName(prefix, symbol string) string {
	symbol = strings.TrimSpace(symbol)
	return prefix + "." + symbol
}

func normalizeFiatForPricing(raw string) string {
	fiat := strings.ToUpper(strings.TrimSpace(raw))
	if fiat == "" {
		return DefaultFiatCurrency
	}
	return fiat
}

func mapPriceValuationError(msg string, err error, meta map[string]string) error {
	grpcCode, ok := grpcCodeFromErrorChain(err)
	if ok {
		meta["grpc_code"] = grpcCode.String()
		switch grpcCode {
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Internal:
			return apperr.PriceUnavailable(msg, err, meta)
		case codes.InvalidArgument, codes.FailedPrecondition, codes.NotFound, codes.PermissionDenied, codes.Unauthenticated:
			return apperr.PriceBadResponse(msg, err, meta)
		default:
			return apperr.PriceBadResponse(msg, err, meta)
		}
	}

	return apperr.PriceUnavailable(msg, err, meta)
}

func grpcCodeFromErrorChain(err error) (codes.Code, bool) {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if st, ok := status.FromError(current); ok {
			return st.Code(), true
		}
	}
	return codes.OK, false
}
