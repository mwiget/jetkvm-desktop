package app

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
)

// Native shell integration. Ebitengine has no mouse, wheel or cursor support
// on mobile, so the iPadOS app forwards pointer input and system state through
// these functions (see mobile/jetkvm). Desktop builds never call them.

// Host button masks use the same bit layout as the HID mouse reports:
// bit 0 primary, bit 1 secondary, bit 2 middle, bits 3 and 4 back and forward.

// HostPointerMoved reports an absolute pointer position in screen points.
func HostPointerMoved(x, y float64) {
	hostInput.push(hostPointerEvent{kind: hostPointerMoveTo, x: x, y: y})
}

// HostPointerMovedBy reports relative motion while the pointer is locked.
func HostPointerMovedBy(dx, dy float64) {
	hostInput.push(hostPointerEvent{kind: hostPointerMoveBy, x: dx, y: dy})
}

// HostPointerButtons reports the full set of currently pressed buttons.
func HostPointerButtons(buttons int) {
	hostInput.push(hostPointerEvent{kind: hostPointerButtons, buttons: byte(buttons)})
}

// HostPointerScrolled reports wheel motion in Ebitengine wheel units.
func HostPointerScrolled(dx, dy float64) {
	hostInput.push(hostPointerEvent{kind: hostPointerScroll, x: dx, y: dy})
}

// HostPointerLockRequested reports whether the app wants relative (captured) mouse input.
func HostPointerLockRequested() bool {
	return hostInput.cursorMode() == ebiten.CursorModeCaptured
}

// HostPointerHidden reports whether the app wants the pointer hidden.
func HostPointerHidden() bool {
	mode := hostInput.cursorMode()
	return mode == ebiten.CursorModeHidden || mode == ebiten.CursorModeCaptured
}

// HostSetDarkMode reports the system appearance.
func HostSetDarkMode(dark bool) {
	if dark {
		hostTheme.Store(int32(themeDark))
	} else {
		hostTheme.Store(int32(themeLight))
	}
}

var (
	hostInput = &hostPointer{}
	hostTheme atomic.Int32
)

// hostSystemTheme returns the appearance reported by a native shell, if any.
func hostSystemTheme() (Theme, bool) {
	theme := Theme(hostTheme.Load())
	return theme, theme != themeUnknown
}

type hostPointerEventKind uint8

const (
	hostPointerMoveTo hostPointerEventKind = iota
	hostPointerMoveBy
	hostPointerButtons
	hostPointerScroll
)

type hostPointerEvent struct {
	kind    hostPointerEventKind
	x, y    float64
	buttons byte
}

// maxPendingHostPointerEvents bounds the queue while ticks are not running,
// e.g. when the game is suspended in the background.
const maxPendingHostPointerEvents = 1024

// hostPointer queues events from the UI thread and applies them at the start
// of each tick, exposing the same pressed / just-pressed / just-released
// semantics as Ebitengine's mouse API.
type hostPointer struct {
	mu      sync.Mutex
	pending []hostPointerEvent
	mode    ebiten.CursorModeType

	x, y           float64
	buttons        byte
	prevButtons    byte
	wheelX, wheelY float64
}

func (p *hostPointer) push(event hostPointerEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pending) >= maxPendingHostPointerEvents && event.kind != hostPointerButtons {
		return
	}
	p.pending = append(p.pending, event)
}

// beginFrame applies queued events. At most one button change is applied per
// tick, so a press and release delivered between two ticks is still observed
// as a press followed by a release.
func (p *hostPointer) beginFrame() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.prevButtons = p.buttons
	p.wheelX, p.wheelY = 0, 0
	buttonsChanged := false
	applied := 0
apply:
	for _, event := range p.pending {
		switch event.kind {
		case hostPointerMoveTo:
			p.x, p.y = event.x, event.y
		case hostPointerMoveBy:
			p.x += event.x
			p.y += event.y
		case hostPointerButtons:
			if event.buttons != p.buttons {
				if buttonsChanged {
					break apply
				}
				p.buttons = event.buttons
				buttonsChanged = true
			}
		case hostPointerScroll:
			p.wheelX += event.x
			p.wheelY += event.y
		}
		applied++
	}
	p.pending = append(p.pending[:0], p.pending[applied:]...)
}

func (p *hostPointer) position() (int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return int(math.Floor(p.x)), int(math.Floor(p.y))
}

func (p *hostPointer) wheel() (float64, float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.wheelX, p.wheelY
}

func hostButtonMask(button ebiten.MouseButton) byte {
	switch button {
	case ebiten.MouseButtonLeft:
		return mouseButtonLeftMask
	case ebiten.MouseButtonRight:
		return mouseButtonRightMask
	case ebiten.MouseButtonMiddle:
		return mouseButtonMiddleMask
	case ebiten.MouseButton3:
		return mouseButtonBackMask
	case ebiten.MouseButton4:
		return mouseButtonForwardMask
	default:
		return 0
	}
}

func (p *hostPointer) pressed(button ebiten.MouseButton) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buttons&hostButtonMask(button) != 0
}

func (p *hostPointer) justPressed(button ebiten.MouseButton) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	mask := hostButtonMask(button)
	return p.buttons&mask != 0 && p.prevButtons&mask == 0
}

func (p *hostPointer) justReleased(button ebiten.MouseButton) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	mask := hostButtonMask(button)
	return p.buttons&mask == 0 && p.prevButtons&mask != 0
}

func (p *hostPointer) setCursorMode(mode ebiten.CursorModeType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = mode
}

func (p *hostPointer) cursorMode() ebiten.CursorModeType {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mode
}
