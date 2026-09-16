package app

import (
	"sync/atomic"
	"time"

	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
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
	hostInactive          atomic.Bool // input goes to another app
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

// HostAppWillResignActive records that input now goes to another app. On iPadOS
// the app can stay on screen meanwhile, beside that app in Split View or Stage
// Manager, so it goes on drawing; but it is no longer the one being typed into,
// so keys held down are let go and input stops, as when a desktop window loses
// focus.
func HostAppWillResignActive() {
	logLifecycle("scene will resign active")
	hostInactive.Store(true)
}

// HostAppDidBecomeActive records that input comes to the app again.
func HostAppDidBecomeActive() {
	logLifecycle("scene did become active")
	hostInactive.Store(false)
}

// HostAppDidEnterBackground records that the app is out of sight. The shell
// stops the game loop at the same time, so nothing is drawn until it comes back
// and the incoming video need not be turned into frames; that is paused here
// rather than through the update loop, which by then is no longer running.
// With reconnectOnReturn it also notes when, so that a return after the system
// has had time to suspend the app replaces the session straight away.
func HostAppDidEnterBackground(reconnectOnReturn bool) {
	logLifecycle("scene did enter background")
	video.SetPaused(true)
	if reconnectOnReturn {
		hostBackgroundedAt.Store(time.Now().UnixNano())
	}
}

// HostAppWillEnterForeground records that the app is coming back into view.
func HostAppWillEnterForeground() {
	logLifecycle("scene will enter foreground")
	video.SetPaused(false)
	hostForegrounded.Store(true)
}

// logLifecycle notes a scene transition, so the device log shows what the app
// was doing around it.
func logLifecycle(msg string) {
	log := logging.Subsystem("app")
	log.Debug().Msg(msg)
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
