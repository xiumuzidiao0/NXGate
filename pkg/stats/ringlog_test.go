package stats

import (
	"fmt"
	"testing"
	"time"
)

func TestRingLogInPlaceRecycle(t *testing.T) {
	capacity := 5
	r := InitRingLog(capacity)

	// Log 10 items into a capacity-5 ring log
	for i := 1; i <= 10; i++ {
		r.Log(LogLevelInfo, "Test", "Message %d", i)
	}

	recent := r.Recent(10)
	if len(recent) != capacity {
		t.Fatalf("Expected %d recent entries, got %d", capacity, len(recent))
	}

	// Verify entries are the latest 5 (6 to 10)
	for i, entry := range recent {
		expectedMsg := fmt.Sprintf("Message %d", i+6)
		if entry.Message != expectedMsg {
			t.Errorf("Expected entry %d to be %q, got %q", i, expectedMsg, entry.Message)
		}
	}
}

func TestRingLogSubscribe(t *testing.T) {
	r := InitRingLog(10)
	ch := r.Subscribe()
	defer r.Unsubscribe(ch)

	r.Log(LogLevelWarning, "SubTest", "Subscribe payload")

	select {
	case entry := <-ch:
		if entry.Message != "Subscribe payload" {
			t.Errorf("Unexpected message: %s", entry.Message)
		}
		if entry.Level != LogLevelWarning {
			t.Errorf("Unexpected level: %s", entry.Level)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for subscribed log entry")
	}
}
