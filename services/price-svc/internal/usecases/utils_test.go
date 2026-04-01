package usecase

import (
	"testing"
	"time"
)

func TestFloorToBucket(t *testing.T) {
	tm := time.Date(2026, 3, 25, 10, 7, 59, 0, time.UTC)
	got := floorToBucket(tm, 5*time.Minute)
	want := time.Date(2026, 3, 25, 10, 5, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("floorToBucket() = %v, want %v", got, want)
	}
}

func TestTruncateDayUTC(t *testing.T) {
	loc := time.FixedZone("UTC+5", 5*60*60)
	tm := time.Date(2026, 3, 25, 23, 30, 10, 0, loc)
	got := truncateDayUTC(tm)
	want := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("truncateDayUTC() = %v, want %v", got, want)
	}
}

func TestNormalizeByOrder(t *testing.T) {
	day := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	prices := [][]float64{
		{1, 100.5},
		{2, 101.5},
	}

	got, err := normalizeByOrder("bitcoin", day, 5*time.Minute, prices)
	if err != nil {
		t.Fatalf("normalizeByOrder() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("normalizeByOrder() len = %d, want 2", len(got))
	}
	if got[0].CoinID != "bitcoin" {
		t.Fatalf("coin id = %q, want bitcoin", got[0].CoinID)
	}
	if !got[0].Time.Equal(day) {
		t.Fatalf("first bucket = %v, want %v", got[0].Time, day)
	}
	if !got[1].Time.Equal(day.Add(5 * time.Minute)) {
		t.Fatalf("second bucket = %v, want %v", got[1].Time, day.Add(5*time.Minute))
	}
}

func TestNormalizeByOrder_InvalidPoint(t *testing.T) {
	day := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	prices := [][]float64{
		{1},
	}

	if _, err := normalizeByOrder("bitcoin", day, 5*time.Minute, prices); err == nil {
		t.Fatal("normalizeByOrder() expected error, got nil")
	}
}
