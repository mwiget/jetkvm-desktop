package app

import "testing"

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
