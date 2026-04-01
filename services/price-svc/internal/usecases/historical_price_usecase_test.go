package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/coingecko"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

type fakeHistoricalRepo struct {
	mu sync.Mutex

	getBatchCalls int
	firstRows     []domain.HistoricalPrice
	alwaysRows    []domain.HistoricalPrice

	stored map[string]domain.HistoricalPrice
	err    error
}

func (r *fakeHistoricalRepo) key(coinID string, t time.Time) string {
	return coinID + "|" + t.UTC().Format(time.RFC3339)
}

func (r *fakeHistoricalRepo) Upsert(context.Context, domain.HistoricalPrice) error {
	return errors.New("not implemented")
}

func (r *fakeHistoricalRepo) Get(context.Context, string, time.Time) (domain.HistoricalPrice, error) {
	return domain.HistoricalPrice{}, errors.New("not implemented")
}

func (r *fakeHistoricalRepo) UpsertBatch(_ context.Context, prices []domain.HistoricalPrice) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stored == nil {
		r.stored = make(map[string]domain.HistoricalPrice)
	}
	for _, p := range prices {
		r.stored[r.key(p.CoinID, p.Time)] = p
	}
	return nil
}

func (r *fakeHistoricalRepo) GetBatch(_ context.Context, keys []domain.PriceKey) ([]domain.HistoricalPrice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.err != nil {
		return nil, r.err
	}

	r.getBatchCalls++
	if r.alwaysRows != nil {
		out := make([]domain.HistoricalPrice, len(r.alwaysRows))
		copy(out, r.alwaysRows)
		return out, nil
	}

	if r.getBatchCalls == 1 && r.firstRows != nil {
		out := make([]domain.HistoricalPrice, len(r.firstRows))
		copy(out, r.firstRows)
		return out, nil
	}

	out := make([]domain.HistoricalPrice, 0, len(keys))
	for _, k := range keys {
		if p, ok := r.stored[r.key(k.CoinID, k.BucketStartUtc)]; ok {
			out = append(out, p)
			continue
		}
		out = append(out, domain.HistoricalPrice{
			CoinID:             k.CoinID,
			Time:               k.BucketStartUtc,
			PriceUsd:           nil,
			GranularitySeconds: nil,
		})
	}
	return out, nil
}

type fakeFXProvider struct {
	calls int
	rate  domain.Fiat
	err   error
}

func (f *fakeFXProvider) Start(context.Context) error { return nil }
func (f *fakeFXProvider) Stop(context.Context) error  { return nil }

func (f *fakeFXProvider) GetUSDtoFiatRate(context.Context, time.Time, string) (domain.Fiat, error) {
	f.calls++
	if f.err != nil {
		return domain.Fiat{}, f.err
	}
	return f.rate, nil
}

func newTestCGClient(t *testing.T, h http.HandlerFunc) *coingecko.CGClient {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	client, err := coingecko.NewCGClient(coingecko.CGConfig{
		BaseURL:         srv.URL,
		RateLimitPerMin: 600,
		GranularityPolicy: coingecko.GranularityPolicy{
			"5minutes": 30 * 24 * time.Hour,
			"1hour":    365 * 24 * time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("NewCGClient() error: %v", err)
	}
	return client
}

func TestGetHistoricalPrices_EmptyFiat(t *testing.T) {
	u := &historicalPriceUC{
		repo:       &fakeHistoricalRepo{},
		fxProvider: &fakeFXProvider{},
		cgClient: newTestCGClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"prices":[]}`))
		}),
	}

	_, err := u.GetHistoricalPrices(context.Background(), " ", []domain.PriceKey{{CoinID: "bitcoin", BucketStartUtc: time.Now()}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestGetHistoricalPrices_EmptyKeys(t *testing.T) {
	u := &historicalPriceUC{
		repo:       &fakeHistoricalRepo{},
		fxProvider: &fakeFXProvider{},
		cgClient: newTestCGClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	got, err := u.GetHistoricalPrices(context.Background(), "USD", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil result, got %v", got)
	}
}

func TestGetHistoricalPrices_USDSkipsFX(t *testing.T) {
	price := decimal.NewFromInt(100)
	g := 300
	keyTime := time.Date(2026, 3, 25, 10, 2, 0, 0, time.UTC)
	bucket := floorToBucket(keyTime, 5*time.Minute)

	repo := &fakeHistoricalRepo{
		alwaysRows: []domain.HistoricalPrice{{
			CoinID:             "bitcoin",
			Time:               bucket,
			PriceUsd:           &price,
			GranularitySeconds: &g,
		}},
	}
	fx := &fakeFXProvider{}

	u := &historicalPriceUC{
		repo:       repo,
		fxProvider: fx,
		cgClient: newTestCGClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	out, err := u.GetHistoricalPrices(context.Background(), " usd ", []domain.PriceKey{{CoinID: "bitcoin", BucketStartUtc: keyTime}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || !out[0].Equal(price) {
		t.Fatalf("unexpected output: %v", out)
	}
	if fx.calls != 0 {
		t.Fatalf("expected no fx calls, got %d", fx.calls)
	}
}

func TestGetHistoricalPrices_NonUSDUsesFX(t *testing.T) {
	price := decimal.NewFromInt(100)
	rate := decimal.NewFromInt(90)
	g := 300
	keyTime := time.Date(2026, 3, 25, 10, 2, 0, 0, time.UTC)
	bucket := floorToBucket(keyTime, 5*time.Minute)

	repo := &fakeHistoricalRepo{
		alwaysRows: []domain.HistoricalPrice{{
			CoinID:             "bitcoin",
			Time:               bucket,
			PriceUsd:           &price,
			GranularitySeconds: &g,
		}},
	}
	fx := &fakeFXProvider{rate: rate}

	u := &historicalPriceUC{
		repo:       repo,
		fxProvider: fx,
		cgClient: newTestCGClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	out, err := u.GetHistoricalPrices(context.Background(), "rub", []domain.PriceKey{{CoinID: "bitcoin", BucketStartUtc: keyTime}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := price.Mul(rate)
	if len(out) != 1 || !out[0].Equal(want) {
		t.Fatalf("unexpected output: %v, want %v", out, want)
	}
	if fx.calls != 1 {
		t.Fatalf("expected 1 fx call, got %d", fx.calls)
	}
}

func TestGetHistoricalPrices_FetchesAndUpsertsMissing(t *testing.T) {
	keyTime := time.Date(2026, 3, 25, 0, 2, 0, 0, time.UTC)
	bucket := floorToBucket(keyTime, 5*time.Minute)

	repo := &fakeHistoricalRepo{
		firstRows: []domain.HistoricalPrice{{
			CoinID:             "bitcoin",
			Time:               bucket,
			PriceUsd:           nil,
			GranularitySeconds: nil,
		}},
	}
	u := &historicalPriceUC{
		repo:       repo,
		fxProvider: &fakeFXProvider{},
		cgClient: newTestCGClient(t, func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/coins/bitcoin/market_chart/range") {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"prices": [][]float64{
					{1, 100},
					{2, 101},
				},
			})
		}),
	}

	out, err := u.GetHistoricalPrices(context.Background(), "USD", []domain.PriceKey{{
		CoinID:         "bitcoin",
		BucketStartUtc: keyTime,
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || !out[0].Equal(decimal.NewFromInt(100)) {
		t.Fatalf("unexpected output: %v", out)
	}
}

func TestGetHistoricalPrices_PriceUnavailableAfterFetch(t *testing.T) {
	keyTime := time.Date(2026, 3, 25, 0, 2, 0, 0, time.UTC)
	bucket := floorToBucket(keyTime, 5*time.Minute)

	repo := &fakeHistoricalRepo{
		alwaysRows: []domain.HistoricalPrice{{
			CoinID:             "bitcoin",
			Time:               bucket,
			PriceUsd:           nil,
			GranularitySeconds: nil,
		}},
	}
	u := &historicalPriceUC{
		repo:       repo,
		fxProvider: &fakeFXProvider{},
		cgClient: newTestCGClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"prices": [][]float64{{1, 100}},
			})
		}),
	}

	_, err := u.GetHistoricalPrices(context.Background(), "USD", []domain.PriceKey{{
		CoinID:         "bitcoin",
		BucketStartUtc: keyTime,
	}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrPriceUnavailable {
		t.Fatalf("expected PRICE_UNAVAILABLE, got %v", err)
	}
}
