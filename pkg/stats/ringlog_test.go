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

func TestRingLogCircularEdgeCases(t *testing.T) {
	r := InitRingLog(5)
	if empty := r.Recent(5); len(empty) != 0 {
		t.Fatalf("expected empty recent slice, got %d", len(empty))
	}

	// Partial fill (3 of 5)
	for i := 1; i <= 3; i++ {
		r.Log(LogLevelInfo, "Test", "M%d", i)
	}
	got := r.Recent(0)
	if len(got) != 3 || got[0].Message != "M1" || got[2].Message != "M3" {
		t.Fatalf("unexpected partial recent: %+v", got)
	}

	// Wrap across end boundary: total 7 items -> buffer holds M3..M7 with head=2
	for i := 4; i <= 7; i++ {
		r.Log(LogLevelInfo, "Test", "M%d", i)
	}
	wrapped := r.Recent(4)
	want := []string{"M4", "M5", "M6", "M7"}
	if len(wrapped) != len(want) {
		t.Fatalf("expected %d entries, got %d", len(want), len(wrapped))
	}
	for i, w := range want {
		if wrapped[i].Message != w {
			t.Fatalf("entry[%d]=%q, want %q", i, wrapped[i].Message, w)
		}
	}
}
