package app

import (
	"testing"
	"time"
)

func TestWakeScheduleWaitsRetriesAndStops(t *testing.T) {
	var w wakeSchedule
	start := time.Unix(1000, 0)

	if _, send := w.due(start, false); send {
		t.Fatal("should not wake immediately after connecting")
	}
	if _, send := w.due(start.Add(wakeFirstDelay/2), false); send {
		t.Fatal("should wait for the first delay")
	}

	now := start.Add(wakeFirstDelay)
	for want := 0; want < wakeMaxAttempts; want++ {
		attempt, send := w.due(now, false)
		if !send || attempt != want {
			t.Fatalf("attempt %d: got attempt=%d send=%v", want, attempt, send)
		}
		if _, send := w.due(now.Add(wakeRetryInterval/2), false); send {
			t.Fatalf("attempt %d: retried before the interval", want)
		}
		now = now.Add(wakeRetryInterval)
	}
	if _, send := w.due(now.Add(time.Hour), false); send {
		t.Fatal("should stop after the maximum number of attempts")
	}
}

func TestWakeScheduleStopsOnceVideoArrives(t *testing.T) {
	var w wakeSchedule
	start := time.Unix(1000, 0)
	w.due(start, false)
	if _, send := w.due(start.Add(wakeFirstDelay), true); send {
		t.Fatal("should not wake when video is already arriving")
	}
	if _, send := w.due(start.Add(time.Minute), false); send {
		t.Fatal("should not resume waking after video arrived")
	}
}
