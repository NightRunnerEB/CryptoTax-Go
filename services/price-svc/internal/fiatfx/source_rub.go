package fiatfx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	inmemory "github.com/NightRunner/CryptoTax-Go/pkg/in-memory"
	applogger "github.com/NightRunner/CryptoTax-Go/pkg/logger"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

const (
	CBRArchiveDailyURL = "https://www.cbr-xml-daily.ru/archive/%s/daily.xml"
	rubCSVHeader       = "Date;USD"
	rubDateFmt         = "02.01.2006"
	rubCSVPath         = "RUB-USD.csv"

	rubReqPerSec = 1
	rubReqPerMin = 30
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
	store      *inmemory.Store[string, Rate]
	schedule   Schedule

	perSecondLimiter *rate.Limiter
	perMinuteLimiter *rate.Limiter

	mu       sync.Mutex
	lastDate time.Time
}

func NewRUBSource(ctx context.Context, httpClient *http.Client) (*RUBSource, error) {
	loc, _ := time.LoadLocation("Europe/Moscow")

	s := &RUBSource{
		httpClient: httpClient,
		store:      inmemory.NewStore[string, Rate](),
		schedule: Schedule{
			Loc:  loc,
			Hour: 20,
			Min:  0,
		},
		perSecondLimiter: rate.NewLimiter(rate.Every(time.Second/time.Duration(rubReqPerSec)), 1),
		perMinuteLimiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(rubReqPerMin)), 1),
	}

	data, last, err := LoadRUBUSDFromCSV(ctx, rubCSVPath, loc)
	if err != nil {
		return nil, err
	}

	if len(data) > 0 && !last.IsZero() {
		s.store.ReplaceAll(data)
		s.lastDate = last
	} else {
		s.lastDate = dateOnly(defaultFromUTC().In(loc), loc).AddDate(0, 0, -1)
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
	var realPoints []csvPoint

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
		realPoints = append(realPoints, csvPoint{Day: day, Rate: usdRate})
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

	if gotReal && len(realPoints) > 0 {
		if err := AppendRUBUSDToCSV(rubCSVPath, realPoints); err != nil {
			log.Error("rub fx append csv failed", zap.Error(err))
		}
	}

	if gotReal && !newLastDate.IsZero() {
		s.mu.Lock()
		s.lastDate = newLastDate
		s.mu.Unlock()
	} else if !gotReal {
		log.Warn(
			"rub fx update had no real points (404 carry-fill only)",
			zap.String("from", from.Format("2006-01-02")),
			zap.String("to", to.Format("2006-01-02")),
		)
	}

	return nil
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

func LoadRUBUSDFromCSV(ctx context.Context, path string, loc *time.Location) (map[string]Rate, time.Time, error) {
	result := make(map[string]Rate)
	var last time.Time
	log := applogger.FromContext(ctx)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn("rub csv file not found, bootstrap from defaultFrom", zap.String("path", path))
			return result, time.Time{}, nil
		}
		return nil, time.Time{}, err
	}
	defer f.Close()

	log.Debug("rub csv read start", zap.String("path", path))

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		if lineNo == 1 {
			line = strings.TrimPrefix(line, "\ufeff")
			if strings.EqualFold(line, rubCSVHeader) {
				continue
			}
		}

		parts := strings.Split(line, ";")
		if len(parts) < 2 {
			continue
		}

		day, err := time.ParseInLocation(rubDateFmt, parts[0], loc)
		if err != nil {
			continue
		}
		day = dateOnly(day, loc)

		rateStr := strings.ReplaceAll(parts[1], ",", ".")
		rate, err := decimal.NewFromString(rateStr)
		if err != nil || rate.IsZero() {
			continue
		}

		key := dateKeyISO(day)
		result[key] = rate

		if last.IsZero() || day.After(last) {
			last = day
		}
	}

	if err := sc.Err(); err != nil {
		return nil, time.Time{}, err
	}

	if !last.IsZero() {
		log.Debug(
			"rub csv read done",
			zap.String("path", path),
			zap.Int("rows", len(result)),
			zap.String("last_day", last.Format("2006-01-02")),
		)
	} else {
		log.Debug("rub csv read done (no rows)", zap.String("path", path))
	}

	return result, last, nil
}

func AppendRUBUSDToCSV(path string, points []csvPoint) error {
	if len(points) == 0 {
		return nil
	}

	var needHeader bool
	if st, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			needHeader = true
		} else {
			return fmt.Errorf("stat csv: %w", err)
		}
	} else {
		needHeader = st.Size() == 0
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open csv for append: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	if needHeader {
		if _, err := w.WriteString("\ufeff" + rubCSVHeader + "\n"); err != nil {
			return fmt.Errorf("write header: %w", err)
		}
	}

	for _, p := range points {
		dayStr := p.Day.Format(rubDateFmt)
		rateStr := strings.ReplaceAll(p.Rate.String(), ".", ",")

		if _, err := w.WriteString(dayStr + ";" + rateStr + "\n"); err != nil {
			return fmt.Errorf("write line: %w", err)
		}
	}

	return nil
}
