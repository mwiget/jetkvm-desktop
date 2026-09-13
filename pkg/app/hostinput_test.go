package app

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestHostPointerQuickClickSpansTwoTicks(t *testing.T) {
	p := &hostPointer{}
	p.push(hostPointerEvent{kind: hostPointerMoveTo, x: 10.7, y: 20.2})
	p.push(hostPointerEvent{kind: hostPointerButtons, buttons: 1})
	p.push(hostPointerEvent{kind: hostPointerButtons, buttons: 0})

	p.beginFrame()
	if x, y := p.position(); x != 10 || y != 20 {
		t.Fatalf("position = %d,%d, want 10,20", x, y)
	}
	if !p.pressed(ebiten.MouseButtonLeft) || !p.justPressed(ebiten.MouseButtonLeft) {
		t.Fatal("first tick should report left pressed and just pressed")
	}

	p.beginFrame()
	if p.pressed(ebiten.MouseButtonLeft) || !p.justReleased(ebiten.MouseButtonLeft) {
		t.Fatal("second tick should report left just released")
	}

	p.beginFrame()
	if p.justReleased(ebiten.MouseButtonLeft) {
		t.Fatal("third tick should not repeat the release")
	}
}

func TestHostPointerButtonMaskMatchesEbitenButtons(t *testing.T) {
	p := &hostPointer{}
	p.push(hostPointerEvent{kind: hostPointerButtons, buttons: 1<<1 | 1<<4})
	p.beginFrame()
	if !p.pressed(ebiten.MouseButtonRight) || !p.pressed(ebiten.MouseButton4) {
		t.Fatal("right and forward buttons should be pressed")
	}
	if p.pressed(ebiten.MouseButtonLeft) || p.pressed(ebiten.MouseButtonMiddle) {
		t.Fatal("left and middle buttons should not be pressed")
	}
	if got := currentMouseButtons(p.pressed); got != mouseButtonRightMask|mouseButtonForwardMask {
		t.Fatalf("currentMouseButtons = %05b", got)
	}
}

func TestHostPointerRelativeMotionAndWheel(t *testing.T) {
	p := &hostPointer{}
	p.push(hostPointerEvent{kind: hostPointerMoveTo, x: 100, y: 100})
	p.push(hostPointerEvent{kind: hostPointerMoveBy, x: -30, y: 5})
	p.push(hostPointerEvent{kind: hostPointerScroll, x: 0.5, y: -1})
	p.push(hostPointerEvent{kind: hostPointerScroll, x: 0.25, y: -2})
	p.beginFrame()
	if x, y := p.position(); x != 70 || y != 105 {
		t.Fatalf("position = %d,%d, want 70,105", x, y)
	}
	if wx, wy := p.wheel(); wx != 0.75 || wy != -3 {
		t.Fatalf("wheel = %v,%v, want 0.75,-3", wx, wy)
	}
	p.beginFrame()
	if wx, wy := p.wheel(); wx != 0 || wy != 0 {
		t.Fatalf("wheel should reset each tick, got %v,%v", wx, wy)
	}
}

func TestHostPointerCursorModeRequests(t *testing.T) {
	saved := hostInput
	t.Cleanup(func() { hostInput = saved })
	hostInput = &hostPointer{}

	hostInput.setCursorMode(ebiten.CursorModeCaptured)
	if !HostPointerLockRequested() || !HostPointerHidden() {
		t.Fatal("captured mode should request lock and hide the pointer")
	}
	hostInput.setCursorMode(ebiten.CursorModeVisible)
	if HostPointerLockRequested() || HostPointerHidden() {
		t.Fatal("visible mode should neither lock nor hide")
	}
}
