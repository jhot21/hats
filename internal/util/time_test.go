package util

import (
	"testing"
	"time"
)

func TestSameDay_SameInstant(t *testing.T) {
	ts := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	if !SameDay(ts, ts) {
		t.Fatal("expected true for same instant")
	}
}

func TestSameDay_SameDayDifferentTimes(t *testing.T) {
	a := time.Date(2024, 3, 15, 8, 0, 0, 0, time.UTC)
	b := time.Date(2024, 3, 15, 23, 59, 59, 0, time.UTC)
	if !SameDay(a, b) {
		t.Fatal("expected true for same calendar day")
	}
}

func TestSameDay_DifferentDays(t *testing.T) {
	a := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	b := time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)
	if SameDay(a, b) {
		t.Fatal("expected false for different days")
	}
}

func TestSameDay_MonthBoundary(t *testing.T) {
	a := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	b := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	if SameDay(a, b) {
		t.Fatal("expected false across month boundary")
	}
}

func TestSameDay_YearBoundary(t *testing.T) {
	a := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	b := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if SameDay(a, b) {
		t.Fatal("expected false across year boundary")
	}
}

func TestSameDay_MidnightBoundary(t *testing.T) {
	a := time.Date(2024, 3, 15, 23, 59, 59, 0, time.UTC)
	b := time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)
	if SameDay(a, b) {
		t.Fatal("expected false at midnight boundary")
	}
}

func TestSameDay_DifferentTimezones(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	// 2024-03-15 01:00 UTC = 2024-03-14 21:00 EDT — different calendar dates
	utc := time.Date(2024, 3, 15, 1, 0, 0, 0, time.UTC)
	ny := utc.In(nyc)
	if SameDay(utc, ny) {
		t.Fatalf("expected false: UTC date is %v, New York date is %v", utc.Format("2006-01-02"), ny.Format("2006-01-02"))
	}
}
