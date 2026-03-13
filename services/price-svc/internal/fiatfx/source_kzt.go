package fiatfx

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	inmemory "github.com/NightRunner/CryptoTax-Go/pkg/in-memory"
	applogger "github.com/NightRunner/CryptoTax-Go/pkg/logger"
)

const (
	NBRKGetRatesURL = "https://nationalbank.kz/rss/get_rates.cfm"
	kztCSVHeader    = "Date;USD"
	kztDateFmt      = "02.01.2006"
	csvPath         = "KZT-USD.csv"
)

type csvPoint struct {
	Day  time.Time
	Rate Rate
}

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
	store      *inmemory.Store[string, Rate]
	schedule   Schedule

	mu       sync.Mutex
	lastDate time.Time
}

func NewKZTSource(ctx context.Context, httpClient *http.Client) (*KZTSource, error) {
	loc, _ := time.LoadLocation("Asia/Almaty")

	s := &KZTSource{
		httpClient: httpClient,
		store:      inmemory.NewStore[string, Rate](),
		schedule: Schedule{
			Loc:  loc,
			Hour: 20,
			Min:  0,
		},
	}

	data, last, err := LoadKZTUSDFromCSV(ctx, csvPath, loc)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("kzt csv file is empty")
	}

	s.store.ReplaceAll(data)
	s.lastDate = last

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

	s.mu.Lock()
	lastSaved := s.lastDate
	s.mu.Unlock()

	from := dateOnly(lastSaved, loc).AddDate(0, 0, 1)
	to := dateOnly(now, loc).AddDate(0, 0, 1)

	if from.After(to) || from.Equal(to) {
		return nil
	}

	var lastCarry Rate
	if r, ok := s.store.Get(dateKeyISO(dateOnly(lastSaved, loc))); ok {
		lastCarry = r
	} // lastSaved всегда должен быть после создания NewKZTSource и чтения из CSV!

	patch := make(map[string]Rate)
	newLastDate := time.Time{}
	gotReal := false
	var realPoints []csvPoint

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		day := dateOnly(d, loc)
		key := dateKeyISO(day)
		url := fmt.Sprintf("%s?fdate=%s", NBRKGetRatesURL, day.Format(kztDateFmt))

		doc, err := FetchXML[nbrkRates](ctx, s.httpClient, url)
		if err != nil {
			log.Warn("kzt fx fetch failed", zap.String("url", url), zap.Error(err))
		} else {
			if usdRate, ok := parseNBRKUSD(doc); ok {
				lastCarry = usdRate
				gotReal = true

				patch[key] = usdRate
				realPoints = append(realPoints, csvPoint{Day: day, Rate: usdRate})
				newLastDate = day
				continue
			}
			log.Warn("kzt fx usd not found in response", zap.String("url", url), zap.Int("items", len(doc.Items)))
		}

		// если не получили курс в этот день, то берем последний известный курс
		patch[dateKeyISO(d)] = lastCarry
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

	if !gotReal {
		log.Error(
			"kzt fx update yielded no rates",
			zap.String("from", from.Format("2006-01-02")),
			zap.String("to", to.Format("2006-01-02")),
		)
		return nil
	}

	if gotReal && len(realPoints) > 0 {
		if err := AppendKZTUSDToCSV(csvPath, realPoints); err != nil {
			log.Error("kzt fx append csv failed", zap.Error(err))
		}
	}

	// lastDate двигаем только если были реальные точки (а не только carry-fill)
	if gotReal && !newLastDate.IsZero() {
		s.mu.Lock()
		s.lastDate = newLastDate
		s.mu.Unlock()
	} else if !gotReal {
		log.Warn(
			"kzt fx update had no real points (carry-fill only)",
			zap.String("from", from.Format("2006-01-02")),
			zap.String("to", to.Format("2006-01-02")),
		)
	}

	return nil
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

func LoadKZTUSDFromCSV(ctx context.Context, path string, loc *time.Location) (map[string]Rate, time.Time, error) {
	result := make(map[string]Rate)
	var last time.Time
	log := applogger.FromContext(ctx)

	f, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer f.Close()

	log.Debug("kzt csv read start", zap.String("path", path))

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
			if strings.EqualFold(line, kztCSVHeader) {
				continue
			}
		}

		parts := strings.Split(line, ";")
		if len(parts) < 2 {
			continue
		}

		day, err := time.ParseInLocation(kztDateFmt, parts[0], loc)
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
			"kzt csv read done",
			zap.String("path", path),
			zap.Int("rows", len(result)),
			zap.String("last_day", last.Format("2006-01-02")),
		)
	} else {
		log.Debug("kzt csv read done (no rows)", zap.String("path", path))
	}

	return result, last, nil
}

// Дубликаты не должны записываться!
func AppendKZTUSDToCSV(path string, points []csvPoint) error {
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
		if _, err := w.WriteString("\ufeff" + kztCSVHeader + "\n"); err != nil {
			return fmt.Errorf("write header: %w", err)
		}
	}

	for _, p := range points {
		dayStr := p.Day.Format(kztDateFmt)

		rateStr := strings.ReplaceAll(p.Rate.String(), ".", ",")

		if _, err := w.WriteString(dayStr + ";" + rateStr + "\n"); err != nil {
			return fmt.Errorf("write line: %w", err)
		}
	}

	return nil
}
