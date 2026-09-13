package hostinput

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestQueueKeepsTextAndKeyTapsInOrder(t *testing.T) {
	q := &queue{}
	q.push(event{text: "ab"})
	q.push(event{text: "c"})
	q.push(event{key: ebiten.KeyBackspace, isKey: true})
	q.push(event{key: ebiten.KeyBackspace, isKey: true})
	q.push(event{text: "d"})

	steps := []struct {
		chars     string
		backspace bool
	}{
		{chars: "abc"},
		{backspace: true},
		{backspace: true},
		{chars: "d"},
		{},
	}
	for i, want := range steps {
		q.beginFrame()
		if got := string(q.appendChars(nil)); got != want.chars {
			t.Fatalf("tick %d: chars = %q, want %q", i, got, want.chars)
		}
		if got := q.tapped(ebiten.KeyBackspace); got != want.backspace {
			t.Fatalf("tick %d: backspace tapped = %v, want %v", i, got, want.backspace)
		}
	}
}

func TestQueueDropsEventsBeyondLimit(t *testing.T) {
	q := &queue{}
	for i := 0; i < maxPendingEvents+10; i++ {
		q.push(event{text: "x"})
	}
	q.beginFrame()
	if got := len(q.appendChars(nil)); got != maxPendingEvents {
		t.Fatalf("delivered %d chars, want %d", got, maxPendingEvents)
	}
}
