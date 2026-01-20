package fiatfx

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	inmemory "github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/in-memory"
	"github.com/shopspring/decimal"
)

const (
	NBRKGetRatesURL = "https://nationalbank.kz/rss/get_rates.cfm"
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
	store      *inmemory.Store[string, Rate]
	schedule   Schedule

	mu       sync.Mutex
	lastDate time.Time
}

func NewKZTSource(httpClient *http.Client) FXSource {
	loc, _ := time.LoadLocation("Asia/Almaty")

	return &KZTSource{
		httpClient: httpClient,
		store:      inmemory.NewStore[string, Rate](),
		schedule: Schedule{
			Loc:  loc,
			Hour: 20,
			Min:  0,
		},
	}
}

func (s *KZTSource) Currency() Currency {
	return KZT
}

func (s *KZTSource) Schedule() Schedule {
	return s.schedule
}

func (s *KZTSource) Get(key time.Time) (Rate, bool) {
	return s.store.Get(dateKeyISO(key))
}

func (s *KZTSource) Update(ctx context.Context) error {
	loc := s.schedule.Loc
	now := time.Now().In(loc)

	s.mu.Lock()
	lastSaved := s.lastDate
	s.mu.Unlock()

	from := lastSaved
	if from.IsZero() {
		from = defaultFrom.In(loc)
	}
	from = dateOnly(from, loc).AddDate(0, 0, 1)
	to := dateOnly(now, loc).AddDate(0, 0, 1)

	if from.After(to) {
		return nil
	}

	var carry Rate
	haveCarry := false
	if !lastSaved.IsZero() {
		if r, ok := s.store.Get(dateKeyISO(dateOnly(lastSaved, loc))); ok {
			carry = r
			haveCarry = true
		}
	}

	patch := make(map[string]Rate)
	newLastDate := time.Time{}

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		url := fmt.Sprintf("%s?fdate=%s", NBRKGetRatesURL, d.Format("02.01.2006"))

		doc, err := FetchXML[nbrkRates](ctx, s.httpClient, url)
		if err == nil {
			usdRate, ok := parseNBRKUSD(doc)
			if ok {
				carry = usdRate
				haveCarry = true
				patch[dateKeyISO(d)] = usdRate
				newLastDate = d
				continue
			}
		}

		// если не получили курс в этот день, то берем последний известный курс
		if haveCarry {
			patch[dateKeyISO(d)] = carry
		}
	}

	if len(patch) == 0 {
		return nil
	}

	s.store.UpsertMany(patch)

	// lastDate двигаем только если были реальные точки (а не только carry-fill).
	if !newLastDate.IsZero() {
		s.mu.Lock()
		s.lastDate = newLastDate
		s.mu.Unlock()
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
