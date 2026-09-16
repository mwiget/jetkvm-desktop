package app

import (
	"sync/atomic"
	"time"

	"github.com/lkarlslund/jetkvm-desktop/pkg/session"
	"github.com/lkarlslund/jetkvm-desktop/pkg/video"
)

// Native shells show an on-screen keyboard while the app wants typing and tell
// the app when it moves between background and foreground. Desktop builds
// never call these.

// hostReconnectAfterBackground is how long the app must have been in the
// background for a return to the foreground to force a reconnect.
const hostReconnectAfterBackground = 3 * time.Second

var (
	hostTextInputActive   atomic.Bool
	hostKeyboardDismissed atomic.Bool
	hostBackgroundedAt    atomic.Int64 // Unix nanoseconds; 0 while in the foreground.
	hostForegrounded      atomic.Bool
)

// HostTextInputActive reports whether the on-screen keyboard should be shown.
func HostTextInputActive() bool {
	return hostTextInputActive.Load()
}

// HostTextInputDismissed records that the user hid the on-screen keyboard. It
// stays hidden until the user focuses a field again.
func HostTextInputDismissed() {
	hostKeyboardDismissed.Store(true)
}

// HostAppWillResignActive records that the app's window is no longer on top.
// The shell stops the game loop at the same time, so nothing will be drawn
// until it comes back and the incoming video need not be turned into frames.
// It pauses the video here rather than through the update loop, which by then
// is no longer running.
func HostAppWillResignActive() {
	video.SetPaused(true)
}

// HostAppDidEnterBackground records when the app moved to the background.
func HostAppDidEnterBackground() {
	hostBackgroundedAt.Store(time.Now().UnixNano())
}

// HostAppDidBecomeActive records that the app returned to the foreground.
func HostAppDidBecomeActive() {
	video.SetPaused(false)
	hostForegrounded.Store(true)
}

// syncHostState publishes keyboard state for the native shell and handles
// lifecycle events it reported since the previous tick.
func (a *App) syncHostState() {
	target := a.hostKeyboardTarget()
	if hostKeyboardDismissed.CompareAndSwap(true, false) {
		a.hostKeyboardDismissedTarget = target
	}
	if target != a.hostKeyboardDismissedTarget {
		a.hostKeyboardDismissedTarget = ""
	}
	hostTextInputActive.Store(target != "" && target != a.hostKeyboardDismissedTarget)

	if hostForegrounded.CompareAndSwap(true, false) {
		backgroundedAt := hostBackgroundedAt.Swap(0)
		switch {
		case backgroundedAt != 0 && time.Since(time.Unix(0, backgroundedAt)) >= hostReconnectAfterBackground:
			a.reconnectAfterBackground()
		case a.ctrl != nil:
			// The session usually survives a short spell out of sight, so show
			// what the screen did while the window was hidden rather than
			// reconnecting over it. It does not survive a long one, and the
			// controller cannot always tell, so check that too.
			a.ctrl.RefreshVideo()
			a.ctrl.ReconnectIfStale()
		}
	}
}

// hostKeyboardTarget identifies what typing would go to. Text fields only
// count once the user tapped them, so the launcher does not open the keyboard
// over the device list on its own.
func (a *App) hostKeyboardTarget() string {
	switch {
	case a.pasteOpen:
		return "paste"
	case a.serialConsoleOpen:
		return "serial"
	}
	if binding := a.currentTextBinding(); binding != nil && binding.ID == a.hostKeyboardField {
		return "field:" + binding.ID
	}
	return ""
}

// reconnectAfterBackground replaces a session that iPadOS likely suspended.
func (a *App) reconnectAfterBackground() {
	if a.ctrl == nil || a.launcherOpen {
		return
	}
	if a.ctrl.Snapshot().Phase == session.PhaseOtherSession {
		return
	}
	a.releaseAllKeys(false)
	a.ctrl.ReconnectNow()
}
