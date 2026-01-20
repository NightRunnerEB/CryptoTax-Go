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
	// CBRDailyURL is not used here (kept for reference). "Daily" endpoint gives only one day.
	CBRDailyURL = "https://www.cbr.ru/scripts/XML_daily.asp"

	// CBRDynamicURL returns a time range of records for a given currency id (VAL_NM_RQ).
	// Important: the response usually contains records only for working days.
	// Weekends/holidays are typically missing and must be handled by the caller.
	CBRDynamicURL = "https://www.cbr.ru/scripts/XML_dynamic.asp"

	// USDValNmRq is the CBR internal id for USD.
	USDValNmRq = "R01235"
)

// defaultFrom is the start point for the very first sync (when lastDate is empty).
// After the first successful run we use lastDate+1 as "from".
// Use a constant to avoid requesting "all history" if lastDate wasn't persisted.
var defaultFrom = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// dynamicValCurs models CBR XML_dynamic.asp response.
type dynamicValCurs struct {
	Records []dynamicRecord `xml:"Record"`
}

// dynamicRecord models a single day record returned by CBR.
// Note: decimal separator is comma in XML (e.g. "77,7586").
type dynamicRecord struct {
	Date      string `xml:"Date,attr"` // "20.01.2026"
	VunitRate string `xml:"VunitRate"` // "77,7586"
}

// RUBSource provides USD/RUB official rate from CBR.
// Storage model:
//   - store key is an ISO day string "YYYY-MM-DD" (see dateKeyISO).
//   - store value is Rate (decimal).
//
// Concurrency:
//   - store is copy-on-write, readers are lock-free (atomic.Value inside).
//   - lastDate is protected by mu.
//
// lastDate semantics:
//   - "the last calendar day we have processed and persisted into store".
//   - we update it only AFTER a successful fetch+parse+persist (so that failures
//     do not move the window forward and do not lose data).
type RUBSource struct {
	httpClient *http.Client
	store      *inmemory.Store[string, Rate]
	schedule   Schedule

	mu       sync.Mutex
	lastDate time.Time
}

func NewRUBSource(httpClient *http.Client) FXSource {
	// We intentionally use Moscow time to match how CBR "days" are interpreted in practice.
	// All internal day arithmetic is done as "date-only" in this location.
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

func (s *RUBSource) Currency() Currency { return RUB }
func (s *RUBSource) Schedule() Schedule { return s.schedule }

// Get returns rate by ISO day key ("YYYY-MM-DD").
func (s *RUBSource) Get(key time.Time) (Rate, bool) { return s.store.Get(dateKeyISO(key)) }

// Update fetches and persists missing days from CBR.
//
// High-level algorithm:
//  1. compute [from..to] where:
//     - from = dateOnly(lastDate)+1 day (or dateOnly(defaultFrom)+1 for first run)
//     - to   = dateOnly(now)
//     If from > to: nothing to do.
//  2. fetch CBR dynamic XML for the range.
//  3. parse records into raw map[day]rate (only valid entries).
//  4. build patch for each day d in [from..to]:
//     - if raw[d] exists: persist it and set carry = raw[d].
//     - else if carry exists: persist carry (carry-forward fill).
//     This is how we handle weekends/holidays: CBR often omits them.
//     - else: skip (can happen on first run if the response has no earlier working day).
//  5. persist patch with UpsertMany (copy-on-write).
//  6. update lastDate ONLY after successful persist.
//
// Notes / gotchas:
//   - We keep store keys in ISO format "YYYY-MM-DD" (NOT the CBR "DD.MM.YYYY").
//     This makes keys stable, sortable, and consistent across sources.
//   - We do not treat "missing day in raw" as an error. For CBR it usually means
//     weekend/holiday, and we fill it by carrying forward the last known working-day rate.
//   - If FetchXML fails, we return an error and do NOT move lastDate forward.
func (s *RUBSource) Update(ctx context.Context) error {
	loc := s.schedule.Loc
	now := time.Now().In(loc)

	s.mu.Lock()
	lastSaved := s.lastDate
	s.mu.Unlock()

	// Compute [from..to] as date-only in the configured location.
	from := lastSaved
	if from.IsZero() {
		from = defaultFrom.In(loc)
	}
	from = dateOnly(from, loc).AddDate(0, 0, 1)
	to := dateOnly(now, loc).AddDate(0, 0, 1)

	if from.After(to) {
		return nil
	}

	// CBR expects DD/MM/YYYY in query params for XML_dynamic.asp.
	url := fmt.Sprintf(
		"%s?date_req1=%s&date_req2=%s&VAL_NM_RQ=%s",
		CBRDynamicURL,
		from.Format("02/01/2006"),
		to.Format("02/01/2006"),
		USDValNmRq,
	)

	doc, err := FetchXML[dynamicValCurs](ctx, s.httpClient, url)
	if err != nil {
		// If request/parsing failed, we do not write anything and do not move lastDate.
		return err
	}
	if len(doc.Records) == 0 {
		// No data returned; keep lastDate unchanged.
		return nil
	}

	// raw contains only valid parsed records keyed by date-only time in loc.
	raw := make(map[time.Time]Rate, len(doc.Records))
	for _, rec := range doc.Records {
		dt, rate, ok := parseCBRRecord(rec, loc)
		if !ok {
			// Skip malformed/empty record.
			continue
		}
		raw[dt] = rate
	}

	// carry is the last known rate from previous successful day, used to fill gaps.
	// We try to load carry from store at lastSaved day (ISO key).
	var carry Rate
	haveCarry := false
	if !lastSaved.IsZero() {
		if r, ok := s.store.Get(dateKeyISO(dateOnly(lastSaved, loc))); ok {
			carry = r
			haveCarry = true
		}
	}

	patch := make(map[string]Rate)
	newLastDate := time.Time{} // will be the last processed day in [from..to] we actually wrote

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if r, ok := raw[d]; ok {
			// Working-day (real) point from CBR.
			carry = r
			haveCarry = true
			patch[dateKeyISO(d)] = r
			newLastDate = d
			continue
		}

		// No record from CBR for day d -> usually weekend/holiday.
		// Fill forward if we have a prior rate.
		if haveCarry {
			patch[dateKeyISO(d)] = carry
			newLastDate = d
		}
		// If we don't have carry (first run and no earlier point), we skip the day.
	}

	if len(patch) == 0 {
		return nil
	}

	// Persist patch atomically (copy-on-write).
	s.store.UpsertMany(patch)

	// Move lastDate only after successful persist.
	s.mu.Lock()
	s.lastDate = newLastDate
	s.mu.Unlock()

	return nil
}

// parseCBRRecord parses a CBR record into (date-only, rate).
// Returns ok=false if record is malformed.
// Note: CBR uses comma as decimal separator.
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
