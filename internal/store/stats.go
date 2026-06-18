package store

import "time"

// Stats holds global slacking-reading statistics, persisted in library.json.
type Stats struct {
	TotalSeconds int    `json:"totalSeconds,omitempty"`
	TodaySeconds int    `json:"todaySeconds,omitempty"`
	LastReadDate string `json:"lastReadDate,omitempty"` // YYYY-MM-DD, local time
	StreakDays   int    `json:"streakDays,omitempty"`
}

// RecordActivity rolls the day/streak on the first activity of a new calendar
// day and adds `seconds` of reading time. Call it on every reading action;
// `seconds` may be 0 (e.g. the first action of a session, or after an idle gap).
func RecordActivity(lib *Library, now time.Time, seconds int) {
	today := now.Format("2006-01-02")
	s := &lib.Stats
	if s.LastReadDate != today {
		switch {
		case s.LastReadDate == "":
			s.StreakDays = 1
		case isPrevDay(s.LastReadDate, today):
			s.StreakDays++
		default:
			s.StreakDays = 1
		}
		s.TodaySeconds = 0
		s.LastReadDate = today
	}
	if seconds > 0 {
		s.TotalSeconds += seconds
		s.TodaySeconds += seconds
	}
}

// isPrevDay reports whether prev is exactly the calendar day before today
// (both "2006-01-02").
func isPrevDay(prev, today string) bool {
	t, err := time.Parse("2006-01-02", today)
	if err != nil {
		return false
	}
	return prev == t.AddDate(0, 0, -1).Format("2006-01-02")
}

// RecordReading advances a book's furthest-read high-water mark. When
// (chapter, para) is beyond the stored furthest position it updates the mark and
// sets CharsRead to charsUpTo (rune count from book start to that position).
// Backward moves and re-reads do not change anything. Unknown id is a no-op.
func RecordReading(lib *Library, id string, chapter, para, charsUpTo int) {
	e := lib.FindByID(id)
	if e == nil {
		return
	}
	if chapter > e.FurthestChapter || (chapter == e.FurthestChapter && para > e.FurthestPara) {
		e.FurthestChapter = chapter
		e.FurthestPara = para
		e.CharsRead = charsUpTo
	}
}
