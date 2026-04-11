package fiatfx

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	inmemory "github.com/NightRunner/CryptoTax-Go/pkg/in-memory"
	applogger "github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
)

const (
	NBRKGetRatesURL = "https://nationalbank.kz/rss/get_rates.cfm"
	kztDateFmt      = "02.01.2006"

	kztLockKey     int64 = 5090101
	kztSourceNBRK        = "nbrk"
	kztSourceCarry       = "carry"
)

var (
	kztFollowWriterPollInterval = 2 * time.Second
	kztFollowWriterMaxWait      = 20 * time.Minute
)

type nbrkRates struct {
	Date  string     `xml:"date"` // "21.01.2026"
	Items []nbrkItem `xml:"item"`
}

type nbrkItem struct {
	Title       string `xml:"title"`       // "USD"
	Description string `xml:"description"` // "471.32"
}

type KZTSource struct {
	httpClient *http.Client
	repo       domain.FXRateRepo
	locker     UpdateLocker
	lockKey    int64
	store      *inmemory.Store[string, Rate]
	schedule   Schedule

	mu       sync.Mutex
	lastDate time.Time
}

func NewKZTSource(
	ctx context.Context,
	httpClient *http.Client,
	repo domain.FXRateRepo,
	locker UpdateLocker,
) (*KZTSource, error) {
	loc, _ := time.LoadLocation("Asia/Almaty")

	s := &KZTSource{
		httpClient: httpClient,
		repo:       repo,
		locker:     locker,
		lockKey:    kztLockKey,
		store:      inmemory.NewStore[string, Rate](),
		schedule: Schedule{
			Loc:  loc,
			Hour: 20,
			Min:  0,
		},
		lastDate: dateOnly(defaultFromUTC().In(loc), loc).AddDate(0, 0, -1),
	}

	loaded, err := s.reloadFromRepo(ctx)
	if err != nil {
		return nil, err
	}
	if !loaded {
		return nil, fmt.Errorf("kzt source bootstrap failed: no fx rates in database")
	}

	return s, nil
}

func (s *KZTSource) Currency() Currency { return KZT }
func (s *KZTSource) Schedule() Schedule { return s.schedule }

func (s *KZTSource) Get(key time.Time) (Rate, bool) {
	loc := s.schedule.Loc
	return s.store.Get(dateKeyISO(dateOnly(key, loc)))
}

func (s *KZTSource) Update(ctx context.Context) error {
	loc := s.schedule.Loc
	now := time.Now().In(loc)
	log := applogger.FromContext(ctx)
	target := dateOnly(now, loc).AddDate(0, 0, 1)

	if _, err := s.reloadFromRepo(ctx); err != nil {
		return err
	}

	unlock, locked, err := s.locker.TryLock(ctx, s.lockKey)
	if err != nil {
		return fmt.Errorf("kzt update lock acquire failed: %w", err)
	}
	if !locked {
		log.Info("kzt fx update skipped: lock not acquired")
		s.waitForWriterAndReload(ctx, target, log)
		return nil
	}
	defer func() {
		if err := unlock(context.Background()); err != nil {
			log.Warn("kzt fx update lock release failed", zap.Error(err))
		}
	}()

	if _, err := s.reloadFromRepo(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	lastSaved := s.lastDate
	s.mu.Unlock()

	from := dateOnly(lastSaved, loc).AddDate(0, 0, 1)
	to := target

	if from.After(to) || from.Equal(to) {
		return nil
	}

	var (
		lastCarry Rate
		haveCarry bool
	)
	if !lastSaved.IsZero() {
		if r, ok := s.store.Get(dateKeyISO(dateOnly(lastSaved, loc))); ok {
			lastCarry = r
			haveCarry = true
		}
	}

	patch := make(map[string]Rate)
	newLastDate := time.Time{}
	gotReal := false

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		day := dateOnly(d, loc)
		key := dateKeyISO(day)
		url := fmt.Sprintf("%s?fdate=%s", NBRKGetRatesURL, day.Format(kztDateFmt))
		isReal := false
		source := kztSourceCarry

		doc, err := FetchXML[nbrkRates](ctx, s.httpClient, url)
		if err != nil {
			log.Warn("kzt fx fetch failed", zap.String("url", url), zap.Error(err))
		} else {
			if usdRate, ok := parseNBRKUSD(doc); ok {
				lastCarry = usdRate
				haveCarry = true
				gotReal = true
				isReal = true
				source = kztSourceNBRK

				patch[key] = usdRate
				if err := s.repo.Upsert(ctx, domain.FXRate{
					Fiat:   KZT,
					Day:    day,
					Rate:   usdRate,
					IsReal: true,
					Source: kztSourceNBRK,
				}); err != nil {
					return fmt.Errorf("kzt upsert rate failed: %w", err)
				}
				newLastDate = day
				continue
			}
			log.Warn("kzt fx usd not found in response", zap.String("url", url), zap.Int("items", len(doc.Items)))
		}

		// если не получили курс в этот день, то берем последний известный курс
		if !haveCarry {
			log.Warn("kzt carry fill skipped: no previous rate", zap.String("day", day.Format(time.DateOnly)))
			continue
		}
		patch[key] = lastCarry
		if err := s.repo.Upsert(ctx, domain.FXRate{
			Fiat:   KZT,
			Day:    day,
			Rate:   lastCarry,
			IsReal: isReal,
			Source: source,
		}); err != nil {
			return fmt.Errorf("kzt upsert carry failed: %w", err)
		}
		newLastDate = day
	}

	if len(patch) == 0 {
		log.Warn(
			"kzt fx update yielded empty patch",
			zap.String("from", from.Format("2006-01-02")),
			zap.String("to", to.Format("2006-01-02")),
		)
		return nil
	}

	s.store.UpsertMany(patch)

	// lastDate двигаем по последней записанной точке (реальной или carry).
	if !newLastDate.IsZero() {
		s.mu.Lock()
		s.lastDate = newLastDate
		s.mu.Unlock()
	}
	if !gotReal {
		log.Warn(
			"kzt fx update had no real points (carry-fill only)",
			zap.String("from", from.Format("2006-01-02")),
			zap.String("to", to.Format("2006-01-02")),
		)
	}

	return nil
}

func (s *KZTSource) waitForWriterAndReload(ctx context.Context, target time.Time, log *zap.Logger) {
	deadline := time.Now().Add(kztFollowWriterMaxWait)
	for {
		loaded, err := s.reloadFromRepo(ctx)
		if err != nil {
			log.Warn("kzt follow-writer reload failed", zap.Error(err))
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
				"kzt follow-writer timeout reached",
				zap.String("target_day", target.Format(time.DateOnly)),
				zap.String("last_day", last.Format(time.DateOnly)),
			)
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(kztFollowWriterPollInterval):
		}
	}
}

func (s *KZTSource) reloadFromRepo(ctx context.Context) (bool, error) {
	rows, err := s.repo.ListByFiat(ctx, KZT)
	if err != nil {
		return false, fmt.Errorf("kzt list rates from repo failed: %w", err)
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

func parseNBRKUSD(doc *nbrkRates) (Rate, bool) {
	for _, it := range doc.Items {
		if strings.TrimSpace(it.Title) != USD {
			continue
		}

		raw := strings.TrimSpace(it.Description)
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
