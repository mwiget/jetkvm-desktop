package app

import (
	"time"

	"github.com/lkarlslund/jetkvm-desktop/pkg/input"
	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
	"github.com/lkarlslund/jetkvm-desktop/pkg/session"
)

// A sleeping computer sends no video, so the stream stays blank after
// connecting until someone presses a key or moves the mouse. Until the first
// frame arrives, send input that wakes most systems without doing anything.

const (
	// wakeFirstDelay gives an awake computer's video a moment to arrive first.
	wakeFirstDelay = time.Second
	// wakeRetryInterval spaces out further attempts while there is no video.
	wakeRetryInterval = 4 * time.Second
	// wakeMaxAttempts bounds how often a connection tries to wake the computer.
	wakeMaxAttempts = 4
)

type wakeSchedule struct {
	attempts int
	nextAt   time.Time
}

// due reports whether to send a wake attempt now, and which attempt it is.
func (w *wakeSchedule) due(now time.Time, hasVideo bool) (attempt int, send bool) {
	if hasVideo {
		w.attempts = wakeMaxAttempts
		return 0, false
	}
	if w.attempts >= wakeMaxAttempts {
		return 0, false
	}
	if w.nextAt.IsZero() {
		w.nextAt = now.Add(wakeFirstDelay)
		return 0, false
	}
	if now.Before(w.nextAt) {
		return 0, false
	}
	attempt = w.attempts
	w.attempts++
	w.nextAt = now.Add(wakeRetryInterval)
	return attempt, true
}

func (a *App) syncWake(now time.Time) {
	if !a.prefs.WakeOnConnect || a.ctrl == nil {
		return
	}
	snap := a.ctrl.Snapshot()
	if snap.Phase != session.PhaseConnected || !snap.HIDReady {
		return
	}
	attempt, send := a.wake.due(now, !a.lastFrameAt.IsZero())
	if !send {
		return
	}
	a.sendWakeInput(attempt)
}

func (a *App) sendWakeInput(attempt int) {
	log := logging.Subsystem("app")
	log.Debug().Int("attempt", attempt+1).Dur("since_connect", a.sinceConnect()).Msg("no video yet, sending wake input")

	// Moving one count right and back leaves the pointer where it was.
	_ = a.ctrl.SendRelMouse(1, 0, 0)
	_ = a.ctrl.SendRelMouse(-1, 0, 0)
	if attempt == 0 {
		return
	}
	// Some systems only wake on keyboard input. Shift alone types nothing.
	if hid, ok := input.KeyToHID(input.KeyShiftLeft); ok {
		_ = a.ctrl.SendKeypress(hid, true)
		_ = a.ctrl.SendKeypress(hid, false)
	}
}
