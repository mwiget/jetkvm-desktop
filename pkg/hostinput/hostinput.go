// Package hostinput merges keyboard input injected by a native shell, such as
// the iPadOS on-screen keyboard, with Ebitengine's own keyboard state.
package hostinput

import (
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// maxPendingEvents bounds the queue while ticks are not running, e.g. when the
// game is suspended in the background.
const maxPendingEvents = 1024

var state = &queue{}

// InsertText queues typed text.
func InsertText(text string) {
	state.push(event{text: text})
}

// TapKey queues a press and release of an editing key such as Backspace or Enter.
func TapKey(key ebiten.Key) {
	state.push(event{key: key, isKey: true})
}

// BeginFrame applies queued input. Call it once at the start of every tick.
func BeginFrame() {
	state.beginFrame()
}

// AppendInputChars is ebiten.AppendInputChars plus text injected this tick.
func AppendInputChars(buf []rune) []rune {
	buf = ebiten.AppendInputChars(buf)
	return state.appendChars(buf)
}

// IsKeyJustPressed is inpututil.IsKeyJustPressed plus keys tapped this tick.
func IsKeyJustPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key) || state.tapped(key)
}

// IsKeyPressed is ebiten.IsKeyPressed plus keys tapped this tick.
func IsKeyPressed(key ebiten.Key) bool {
	return ebiten.IsKeyPressed(key) || state.tapped(key)
}

type event struct {
	text  string
	key   ebiten.Key
	isKey bool
}

type queue struct {
	mu      sync.Mutex
	pending []event
	chars   []rune
	keys    []ebiten.Key
}

func (q *queue) push(e event) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) >= maxPendingEvents {
		return
	}
	q.pending = append(q.pending, e)
}

// beginFrame delivers either all text queued before the next key tap, or a
// single key tap. Each tap therefore gets its own tick and is seen as a
// separate just-pressed event, and text and keys stay in order.
func (q *queue) beginFrame() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.chars = q.chars[:0]
	q.keys = q.keys[:0]
	applied := 0
	for _, e := range q.pending {
		if e.isKey {
			if applied == 0 {
				q.keys = append(q.keys, e.key)
				applied++
			}
			break
		}
		q.chars = append(q.chars, []rune(e.text)...)
		applied++
	}
	q.pending = append(q.pending[:0], q.pending[applied:]...)
}

func (q *queue) appendChars(buf []rune) []rune {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append(buf, q.chars...)
}

func (q *queue) tapped(key ebiten.Key) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return slices.Contains(q.keys, key)
}
