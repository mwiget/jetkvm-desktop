package app

import (
	"strings"

	"github.com/lkarlslund/jetkvm-desktop/pkg/discovery"
	"github.com/lkarlslund/jetkvm-desktop/pkg/session"
)

// JetKVM devices only report their hostname after login, and discovery can
// usually only name them by IP. Remember the hostname per base URL once
// connected, and show it in the launcher from then on.

// withSavedDeviceName names a discovered device after the hostname saved for
// it, unless reverse DNS already provided a name.
func (a *App) withSavedDeviceName(device discovery.Device) discovery.Device {
	if device.Host != "" {
		return device
	}
	if name := a.prefs.DeviceNames[device.BaseURL]; name != "" {
		device.Name = name
	}
	return device
}

// rememberDeviceName saves the connected device's hostname when it changes.
func (a *App) rememberDeviceName(snap session.Snapshot) {
	if snap.Phase != session.PhaseConnected {
		return
	}
	name := strings.TrimSpace(snap.Hostname)
	baseURL := a.cfg.BaseURL
	if name == "" || baseURL == "" || a.prefs.DeviceNames[baseURL] == name {
		return
	}
	if a.prefs.DeviceNames == nil {
		a.prefs.DeviceNames = map[string]string{}
	}
	a.prefs.DeviceNames[baseURL] = name
	a.savePreferences()

	for i := range a.discovered {
		if a.discovered[i].BaseURL == baseURL {
			a.discovered[i] = a.withSavedDeviceName(a.discovered[i])
		}
	}
	a.sortDiscovered()
}
