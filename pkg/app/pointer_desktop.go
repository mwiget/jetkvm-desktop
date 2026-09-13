//go:build !ios

package app

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// platformHasWindow reports whether the app runs in a desktop window that can
// be resized, moved and made fullscreen.
const platformHasWindow = true

func beginPointerFrame() {}

func cursorPosition() (int, int) { return ebiten.CursorPosition() }

func isMouseButtonPressed(button ebiten.MouseButton) bool {
	return ebiten.IsMouseButtonPressed(button)
}

func isMouseButtonJustPressed(button ebiten.MouseButton) bool {
	return inpututil.IsMouseButtonJustPressed(button)
}

func isMouseButtonJustReleased(button ebiten.MouseButton) bool {
	return inpututil.IsMouseButtonJustReleased(button)
}

func mouseWheel() (float64, float64) { return ebiten.Wheel() }

func setCursorMode(mode ebiten.CursorModeType) { ebiten.SetCursorMode(mode) }

func isFullscreen() bool { return ebiten.IsFullscreen() }

func setFullscreen(fullscreen bool) { ebiten.SetFullscreen(fullscreen) }
