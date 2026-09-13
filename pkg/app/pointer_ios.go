//go:build ios

package app

import "github.com/hajimehoshi/ebiten/v2"

// platformHasWindow reports whether the app runs in a desktop window that can
// be resized, moved and made fullscreen.
const platformHasWindow = false

func beginPointerFrame() { hostInput.beginFrame() }

func cursorPosition() (int, int) { return hostInput.position() }

func isMouseButtonPressed(button ebiten.MouseButton) bool { return hostInput.pressed(button) }

func isMouseButtonJustPressed(button ebiten.MouseButton) bool {
	return hostInput.justPressed(button)
}

func isMouseButtonJustReleased(button ebiten.MouseButton) bool {
	return hostInput.justReleased(button)
}

func mouseWheel() (float64, float64) { return hostInput.wheel() }

func setCursorMode(mode ebiten.CursorModeType) { hostInput.setCursorMode(mode) }

func isFullscreen() bool { return false }

func setFullscreen(bool) {}
