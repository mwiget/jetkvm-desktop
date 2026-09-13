package app

import (
	"time"

	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
	"github.com/lkarlslund/jetkvm-desktop/pkg/session"
)

// Debug-level connection timeline: how long each phase, status change (such
// as the device's video input state) and the first video frame take after a
// connection starts.

func (a *App) sinceConnect() time.Duration {
	if a.connectStartedAt.IsZero() {
		return 0
	}
	return time.Since(a.connectStartedAt)
}

func (a *App) logConnectionProgress(snap session.Snapshot) {
	if snap.Phase == a.loggedPhase && snap.Status == a.loggedStatus {
		return
	}
	a.loggedPhase = snap.Phase
	a.loggedStatus = snap.Status
	log := logging.Subsystem("app")
	log.Debug().
		Str("phase", snap.Phase.String()).
		Str("status", snap.Status).
		Dur("since_connect", a.sinceConnect()).
		Msg("connection progress")
}

func (a *App) logFirstVideoFrame() {
	log := logging.Subsystem("app")
	log.Debug().Dur("since_connect", a.sinceConnect()).Msg("first video frame")
}
