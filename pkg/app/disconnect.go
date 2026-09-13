package app

import (
	"time"

	"github.com/lkarlslund/jetkvm-desktop/pkg/client"
	"github.com/lkarlslund/jetkvm-desktop/pkg/session"
)

// disconnectToLauncher ends the current session and returns to the device
// list, so another device can be chosen without restarting the app.
func (a *App) disconnectToLauncher() {
	a.settingsOpen = false
	a.pasteOpen = false
	a.mediaOpen = false
	a.serialConsoleOpen = false
	a.statsOpen = false
	a.relative = false

	if a.ctrl != nil {
		a.releaseAllKeys(true)
		a.ctrl.Stop()
		a.ctrl = nil
	}

	// Forget the device and its password; the next device may need a different one.
	a.cfg.BaseURL = ""
	a.cfg.Password = ""
	a.pendingTarget = ""
	a.launcherPassword = ""
	a.launcherError = ""
	a.launcherMode = launcherModeBrowse
	a.launcherOpen = true
	a.allowDiscovery = true

	a.mu.Lock()
	if a.lastImg != nil {
		a.lastImg.Deallocate()
		a.lastImg = nil
	}
	a.lastFrameAt = time.Time{}
	a.mu.Unlock()
	a.lastPhase = session.PhaseIdle
	a.resetConnectionHardwareState()
	a.stats = client.StatsSnapshot{}
	a.statsHistory = nil

	a.applyCursorMode()
}
