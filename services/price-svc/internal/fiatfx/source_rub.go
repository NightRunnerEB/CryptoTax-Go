package fiatfx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	inmemory "github.com/NightRunner/CryptoTax-Go/pkg/in-memory"
	applogger "github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

const (
	CBRArchiveDailyURL = "https://www.cbr-xml-daily.ru/archive/%s/daily.xml"

	rubReqPerSec = 1
	rubReqPerMin = 30

	rubLockKey     int64 = 5090102
	rubSourceCBR         = "cbr"
	rubSourceCarry       = "carry"
)

var (
	rubFollowWriterPollInterval = 2 * time.Second
	rubFollowWriterMaxWait      = 20 * time.Minute
)

type cbrDaily struct {
	Date  string         `xml:"Date,attr"` // "19.01.2024"
	Items []cbrDailyItem `xml:"Valute"`
}

type cbrDailyItem struct {
	CharCode  string `xml:"CharCode"`  // "USD"
	Value     string `xml:"Value"`     // "88,5896"
	VunitRate string `xml:"VunitRate"` // "88,5896"
}

type RUBSource struct {
	httpClient *http.Client
	repo       domain.FXRateRepo
	locker     UpdateLocker
	lockKey    int64
	store      *inmemory.Store[string, Rate]
	schedule   Schedule

	perSecondLimiter *rate.Limiter
	perMinuteLimiter *rate.Limiter

	mu       sync.Mutex
	lastDate time.Time
}

func NewRUBSource(
	ctx context.Context,
	httpClient *http.Client,
	repo domain.FXRateRepo,
	locker UpdateLocker,
) (*RUBSource, error) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	if httpClient == nil {
		return nil, fmt.Errorf("rub source http client is nil")
	}
	if repo == nil {
		return nil, fmt.Errorf("rub source fx repo is nil")
	}
	if locker == nil {
		return nil, fmt.Errorf("rub source update locker is nil")
	}

	s := &RUBSource{
		httpClient: httpClient,
		repo:       repo,
		locker:     locker,
		lockKey:    rubLockKey,
		store:      inmemory.NewStore[string, Rate](),
		schedule: Schedule{
			Loc:  loc,
			Hour: 20,
			Min:  0,
		},
		perSecondLimiter: rate.NewLimiter(rate.Every(time.Second/time.Duration(rubReqPerSec)), 1),
		perMinuteLimiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(rubReqPerMin)), 1),
		lastDate:         dateOnly(defaultFromUTC().In(loc), loc).AddDate(0, 0, -1),
	}

	loaded, err := s.reloadFromRepo(ctx)
	if err != nil {
		return nil, err
	}
	if !loaded {
		return nil, fmt.Errorf("rub source bootstrap failed: no fx rates in database")
	}

	return s, nil
}

func (s *RUBSource) Currency() Currency { return RUB }
func (s *RUBSource) Schedule() Schedule { return s.schedule }
func (s *RUBSource) Get(key time.Time) (Rate, bool) {
	loc := s.schedule.Loc
	return s.store.Get(dateKeyISO(dateOnly(key, loc)))
}

func (s *RUBSource) Update(ctx context.Context) error {
	loc := s.schedule.Loc
	now := time.Now().In(loc)
	log := applogger.FromContext(ctx)
	target := dateOnly(now, loc)

	if _, err := s.reloadFromRepo(ctx); err != nil {
		return err
	}

	unlock, locked, err := s.locker.TryLock(ctx, s.lockKey)
	if err != nil {
		return fmt.Errorf("rub update lock acquire failed: %w", err)
	}
	if !locked {
		log.Info("rub fx update skipped: lock not acquired")
		s.waitForWriterAndReload(ctx, target, log)
		return nil
	}
	defer func() {
		if err := unlock(context.Background()); err != nil {
			log.Warn("rub fx update lock release failed", zap.Error(err))
		}
	}()

	if _, err := s.reloadFromRepo(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	lastSaved := s.lastDate
	s.mu.Unlock()

	from := dateOnly(lastSaved, loc).AddDate(0, 0, 1)
	to := dateOnly(now, loc)

	if from.After(to) {
		return nil
	}

	patch := make(map[string]Rate)
	newLastDate := time.Time{}
	gotReal := false

	var carry Rate
	haveCarry := false
	if r, ok := s.store.Get(dateKeyISO(dateOnly(lastSaved, loc))); ok {
		carry = r
		haveCarry = true
	}

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		day := dateOnly(d, loc)
		key := dateKeyISO(day)
		url := fmt.Sprintf(CBRArchiveDailyURL, day.Format("2006/01/02"))

		if err := s.waitRateLimit(ctx); err != nil {
			return err
		}

		doc, err := FetchXML[cbrDaily](ctx, s.httpClient, url)
		if err != nil {
			if isHTTPStatus(err, http.StatusNotFound) {
				if !haveCarry {
					bootstrapCarry, ok, bootErr := s.bootstrapCarry(ctx, day, log)
					if bootErr != nil {
						return bootErr
					}
					if ok {
						carry = bootstrapCarry
						haveCarry = true
					}
				}

				if haveCarry {
					patch[key] = carry
					if err := s.repo.Upsert(ctx, domain.FXRate{
						Fiat:   RUB,
						Day:    day,
						Rate:   carry,
						IsReal: false,
						Source: rubSourceCarry,
					}); err != nil {
						return fmt.Errorf("rub upsert carry failed: %w", err)
					}
					newLastDate = day
				} else {
					log.Warn(
						"rub fx carry fill skipped: no previous rate",
						zap.String("day", day.Format("2006-01-02")),
						zap.String("url", url),
					)
				}
				continue
			}

			log.Warn("rub fx fetch failed", zap.String("url", url), zap.Error(err))
			return err
		}

		usdRate, ok := parseCBRUSD(doc)
		if !ok {
			log.Warn("rub fx usd not found in response", zap.String("url", url), zap.Int("items", len(doc.Items)))
			return fmt.Errorf("rub fx usd not found in response: %s", url)
		}

		carry = usdRate
		haveCarry = true
		gotReal = true

		patch[key] = usdRate
		if err := s.repo.Upsert(ctx, domain.FXRate{
			Fiat:   RUB,
			Day:    day,
			Rate:   usdRate,
			IsReal: true,
			Source: rubSourceCBR,
		}); err != nil {
			return fmt.Errorf("rub upsert rate failed: %w", err)
		}
		newLastDate = day
	}

	if len(patch) == 0 {
		log.Warn(
			"rub fx update yielded empty patch",
			zap.String("from", from.Format("2006-01-02")),
			zap.String("to", to.Format("2006-01-02")),
		)
		return nil
	}

	s.store.UpsertMany(patch)

	if !newLastDate.IsZero() {
		s.mu.Lock()
		s.lastDate = newLastDate
		s.mu.Unlock()
	}
	if !gotReal {
		log.Warn(
			"rub fx update had no real points (404 carry-fill only)",
			zap.String("from", from.Format("2006-01-02")),
			zap.String("to", to.Format("2006-01-02")),
		)
	}

	return nil
}

func (s *RUBSource) waitForWriterAndReload(ctx context.Context, target time.Time, log *zap.Logger) {
	deadline := time.Now().Add(rubFollowWriterMaxWait)
	for {
		loaded, err := s.reloadFromRepo(ctx)
		if err != nil {
			log.Warn("rub follow-writer reload failed", zap.Error(err))
			return
		}
		s.mu.Lock()
		last := s.lastDate
		s.mu.Unlock()
		if loaded && !last.Before(target) {
			return
		}
		if time.Now().After(deadline) {
			log.Warn(
				"rub follow-writer timeout reached",
				zap.String("target_day", target.Format(time.DateOnly)),
				zap.String("last_day", last.Format(time.DateOnly)),
			)
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(rubFollowWriterPollInterval):
		}
	}
}

func (s *RUBSource) reloadFromRepo(ctx context.Context) (bool, error) {
	rows, err := s.repo.ListByFiat(ctx, RUB)
	if err != nil {
		return false, fmt.Errorf("rub list rates from repo failed: %w", err)
	}
	if len(rows) == 0 {
		return false, nil
	}

	loc := s.schedule.Loc
	data := make(map[string]Rate, len(rows))
	var last time.Time
	for _, row := range rows {
		day := dateOnly(row.Day, loc)
		key := dateKeyISO(day)
		data[key] = row.Rate
		if last.IsZero() || day.After(last) {
			last = day
		}
	}
	s.store.ReplaceAll(data)
	if !last.IsZero() {
		s.mu.Lock()
		s.lastDate = last
		s.mu.Unlock()
	}
	return true, nil
}

func (s *RUBSource) waitRateLimit(ctx context.Context) error {
	if err := s.perSecondLimiter.Wait(ctx); err != nil {
		return apperr.Internal("rub per-second rate limit wait failed", err, nil)
	}
	if err := s.perMinuteLimiter.Wait(ctx); err != nil {
		return apperr.Internal("rub per-minute rate limit wait failed", err, nil)
	}
	return nil
}

func isHTTPStatus(err error, statusCode int) bool {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}

	status := strings.TrimSpace(appErr.Meta["status"])
	if status == "" {
		return false
	}
	return strings.HasPrefix(status, fmt.Sprintf("%d", statusCode))
}

func parseCBRUSD(doc *cbrDaily) (Rate, bool) {
	for _, it := range doc.Items {
		if strings.TrimSpace(it.CharCode) != USD {
			continue
		}

		raw := strings.TrimSpace(it.VunitRate)
		if raw == "" {
			raw = strings.TrimSpace(it.Value)
		}
		if raw == "" {
			return Rate{}, false
		}
		raw = strings.ReplaceAll(raw, ",", ".")

		val, err := decimal.NewFromString(raw)
		if err != nil || val.IsZero() {
			return Rate{}, false
		}

		return val, true
	}

	return Rate{}, false
}

func (s *RUBSource) bootstrapCarry(ctx context.Context, day time.Time, log *zap.Logger) (Rate, bool, error) {
	const maxLookbackDays = 14

	for i := 1; i <= maxLookbackDays; i++ {
		prevDay := day.AddDate(0, 0, -i)
		url := fmt.Sprintf(CBRArchiveDailyURL, prevDay.Format("2006/01/02"))

		if err := s.waitRateLimit(ctx); err != nil {
			return Rate{}, false, err
		}

		doc, err := FetchXML[cbrDaily](ctx, s.httpClient, url)
		if err != nil {
			if isHTTPStatus(err, http.StatusNotFound) {
				continue
			}
			log.Warn("rub fx bootstrap carry fetch failed", zap.String("url", url), zap.Error(err))
			return Rate{}, false, err
		}

		usdRate, ok := parseCBRUSD(doc)
		if !ok {
			log.Warn("rub fx bootstrap carry usd not found", zap.String("url", url), zap.Int("items", len(doc.Items)))
			return Rate{}, false, fmt.Errorf("rub fx bootstrap carry usd not found: %s", url)
		}

		return usdRate, true, nil
	}

	return Rate{}, false, nil
}
