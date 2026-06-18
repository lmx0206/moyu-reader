package store

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestRecordActivityStreak(t *testing.T) {
	lib := NewLibrary()
	// first ever day -> streak 1
	RecordActivity(lib, day("2026-06-18"), 60)
	if lib.Stats.StreakDays != 1 || lib.Stats.TotalSeconds != 60 || lib.Stats.TodaySeconds != 60 {
		t.Fatalf("day1: %+v", lib.Stats)
	}
	// same day again -> no streak change, time accumulates
	RecordActivity(lib, day("2026-06-18"), 30)
	if lib.Stats.StreakDays != 1 || lib.Stats.TodaySeconds != 90 {
		t.Fatalf("same day: %+v", lib.Stats)
	}
	// next consecutive day -> streak 2, today resets then adds
	RecordActivity(lib, day("2026-06-19"), 10)
	if lib.Stats.StreakDays != 2 || lib.Stats.TodaySeconds != 10 || lib.Stats.TotalSeconds != 100 {
		t.Fatalf("day2: %+v", lib.Stats)
	}
	// skip a day -> streak resets to 1
	RecordActivity(lib, day("2026-06-21"), 5)
	if lib.Stats.StreakDays != 1 || lib.Stats.TodaySeconds != 5 {
		t.Fatalf("gap: %+v", lib.Stats)
	}
}

func TestRecordReadingHighWater(t *testing.T) {
	lib := NewLibrary()
	lib.Books = append(lib.Books, BookEntry{ID: "x"})
	RecordReading(lib, "x", 0, 3, 100)
	if e := lib.FindByID("x"); e.FurthestPara != 3 || e.CharsRead != 100 {
		t.Fatalf("advance: %+v", e)
	}
	// going backward does NOT lower the high-water mark
	RecordReading(lib, "x", 0, 1, 40)
	if e := lib.FindByID("x"); e.FurthestPara != 3 || e.CharsRead != 100 {
		t.Fatalf("backward should not change: %+v", e)
	}
	// advancing into a later chapter updates it
	RecordReading(lib, "x", 1, 0, 150)
	if e := lib.FindByID("x"); e.FurthestChapter != 1 || e.FurthestPara != 0 || e.CharsRead != 150 {
		t.Fatalf("next chapter: %+v", e)
	}
	// unknown id is a no-op (no panic)
	RecordReading(lib, "nope", 9, 9, 999)
}
