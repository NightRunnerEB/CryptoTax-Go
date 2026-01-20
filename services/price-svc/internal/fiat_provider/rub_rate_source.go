package resolve

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
	CBRDailyURL   = "https://www.cbr.ru/scripts/XML_daily.asp"
	CBRDynamicURL = "https://www.cbr.ru/scripts/XML_dynamic.asp"
	USDValNmRq    = "R01235"
)

var defaultFrom = time.Date(2013, 1, 1, 0, 0, 0, 0, time.UTC)

type dynamicValCurs struct {
	Records []dynamicRecord `xml:"Record"`
}

type dynamicRecord struct {
	Date      string `xml:"Date,attr"` // "20.01.2026"
	VunitRate string `xml:"VunitRate"` // "77,7586"
}

type RUBSource struct {
	httpClient *http.Client
	store      *inmemory.Store[string, Rate]
	schedule   Schedule

	mu       sync.Mutex
	lastDate time.Time
}

func NewRUBSource(httpClient *http.Client) FXSource {
	loc, _ := time.LoadLocation("Europe/Moscow")

	return &RUBSource{
		httpClient: httpClient,
		store:      inmemory.NewStore[string, Rate](),
		schedule: Schedule{
			Loc:  loc,
			Hour: 20,
			Min:  0,
		},
	}
}

func (s *RUBSource) Currency() Currency {
	return RUB
}

func (s *RUBSource) Schedule() Schedule {
	return s.schedule
}

func (s *RUBSource) Get(key string) (Rate, bool) {
	return s.store.Get(key)
}

func (s *RUBSource) Update(ctx context.Context) error {
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
	to := dateOnly(now, loc)

	if from.After(to) {
		return nil
	}

	url := fmt.Sprintf(
		"%s?date_req1=%s&date_req2=%s&VAL_NM_RQ=%s",
		CBRDynamicURL,
		from.Format("02/01/2006"),
		to.Format("02/01/2006"),
		USDValNmRq,
	)

	doc, err := FetchXML[dynamicValCurs](ctx, s.httpClient, url)
	if err != nil {
		return err
	}

	if len(doc.Records) == 0 {
		return nil
	}

	raw := make(map[time.Time]Rate, len(doc.Records))
	for _, rec := range doc.Records {
		dt, rate, ok := parseCBRRecord(rec, loc)
		if !ok {
			continue
		}
		raw[dt] = rate
	}

	// Last saved rate to carry forward
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
		if r, ok := raw[d]; ok {
			carry = r
			haveCarry = true
			patch[dateKeyISO(d)] = r
			newLastDate = d
			continue
		}

		// if no rate for the day, but we have carry from previous days
		if haveCarry {
			patch[dateKeyISO(d)] = carry
		}
		// else no rate and no carry - skip
	}

	if len(patch) == 0 {
		return nil
	}

	s.store.UpsertMany(patch)

	s.mu.Lock()
	s.lastDate = newLastDate
	s.mu.Unlock()

	return nil
}

func parseCBRRecord(rec dynamicRecord, loc *time.Location) (time.Time, Rate, bool) {
	if strings.TrimSpace(rec.Date) == "" {
		return time.Time{}, Rate{}, false
	}

	dt, err := time.ParseInLocation("02.01.2006", strings.TrimSpace(rec.Date), loc)
	if err != nil {
		return time.Time{}, Rate{}, false
	}
	dt = dateOnly(dt, loc)

	raw := strings.TrimSpace(strings.ReplaceAll(rec.VunitRate, ",", "."))
	if raw == "" {
		return time.Time{}, Rate{}, false
	}

	rate, err := decimal.NewFromString(raw)
	if err != nil || rate.IsZero() {
		return time.Time{}, Rate{}, false
	}

	return dt, rate, true
}
