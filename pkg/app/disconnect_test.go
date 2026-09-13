package app

import (
	"testing"

	"github.com/lkarlslund/jetkvm-desktop/pkg/session"
)

// Regression: the disconnect button cleared the session controller from its
// click handler, and the rest of that Update dereferenced it (nil panic that
// stopped the game loop on iPad).
func TestDisconnectButtonDefersUntilNextTick(t *testing.T) {
	a, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	a.ctrl = session.New(session.Config{BaseURL: "http://192.168.1.50"})

	var clicked bool
	for _, button := range a.layoutChromeButtons(1280, 720, session.Snapshot{Phase: session.PhaseConnected}) {
		if button.id == "disconnect" {
			button.onClick()
			clicked = true
		}
	}
	if !clicked {
		t.Fatal("disconnect button not found")
	}
	if a.ctrl == nil || !a.disconnectRequested {
		t.Fatal("clicking disconnect must only request it, keeping the controller for the rest of the tick")
	}

	a.applyPendingDisconnect()
	if a.ctrl != nil || a.disconnectRequested || !a.launcherOpen {
		t.Fatal("pending disconnect should be applied at the start of the next tick")
	}
}

func TestDisconnectToLauncherResetsSessionState(t *testing.T) {
	a := &App{
		cfg:               Config{BaseURL: "http://192.168.1.50", Password: "secret"},
		launcherOpen:      false,
		launcherMode:      launcherModePassword,
		launcherPassword:  "secret",
		launcherError:     "Authentication failed",
		pendingTarget:     "http://192.168.1.50",
		settingsOpen:      true,
		pasteOpen:         true,
		mediaOpen:         true,
		serialConsoleOpen: true,
		statsOpen:         true,
		relative:          true,
	}

	a.disconnectToLauncher()

	if !a.launcherOpen || a.launcherMode != launcherModeBrowse || !a.allowDiscovery {
		t.Fatalf("launcher not reopened for browsing: open=%v mode=%v discovery=%v", a.launcherOpen, a.launcherMode, a.allowDiscovery)
	}
	if a.cfg.BaseURL != "" || a.cfg.Password != "" || a.launcherPassword != "" || a.pendingTarget != "" || a.launcherError != "" {
		t.Fatalf("device details not cleared: %+v password=%q target=%q error=%q", a.cfg, a.launcherPassword, a.pendingTarget, a.launcherError)
	}
	if a.settingsOpen || a.pasteOpen || a.mediaOpen || a.serialConsoleOpen || a.statsOpen || a.relative {
		t.Fatal("overlays or relative mouse mode left active")
	}
	if a.ctrl != nil {
		t.Fatal("session controller not cleared")
	}
	if !a.shouldRunDiscovery() {
		t.Fatal("discovery should run after returning to the launcher")
	}
}
