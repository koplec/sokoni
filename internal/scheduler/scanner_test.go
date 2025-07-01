package scheduler

import (
	"os"
	"testing"
	"time"
)

func TestNewScannerDefaultInterval(t *testing.T) {
	os.Unsetenv("SOKONI_SCHEDULER_INTERVAL")
	s := NewScanner(nil)
	if s.interval != 6*time.Hour {
		t.Errorf("expected default 6h, got %v", s.interval)
	}
}

func TestNewScannerCustomInterval(t *testing.T) {
	os.Setenv("SOKONI_SCHEDULER_INTERVAL", "1h30m")
	defer os.Unsetenv("SOKONI_SCHEDULER_INTERVAL")
	s := NewScanner(nil)
	want := time.Hour + 30*time.Minute
	if s.interval != want {
		t.Errorf("expected %v, got %v", want, s.interval)
	}
}
