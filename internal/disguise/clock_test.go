package disguise

import (
	"testing"
	"time"
)

func TestLogClockFormatAndIncrement(t *testing.T) {
	base := time.Date(2026, 6, 12, 14, 23, 1, 0, time.UTC)
	if got := logClock(base, 0); got != "14:23:01" {
		t.Fatalf("logClock(base,0) = %q want 14:23:01", got)
	}
	if got := logClock(base, 5); got != "14:23:06" {
		t.Fatalf("logClock(base,5) = %q want 14:23:06", got)
	}
}

func TestLogPrefixUsesClock(t *testing.T) {
	// With a deterministic base, the prefix must contain that time.
	old := clockBase
	clockBase = time.Date(2026, 6, 12, 9, 30, 0, 0, time.UTC)
	defer func() { clockBase = old }()
	if got := (logTheme{}).LinePrefix(0); got[:10] != "[09:30:00]" {
		t.Fatalf("prefix should start with clock time, got %q", got)
	}
}
