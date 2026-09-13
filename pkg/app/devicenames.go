package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/lkarlslund/jetkvm-desktop/pkg/discovery"
	"github.com/lkarlslund/jetkvm-desktop/pkg/session"
)

// JetKVM devices only report their hostname, ID and version after login, and
// discovery can usually only name them by IP. Remember what a connected device
// reports, and show it in the launcher from then on.

// savedDeviceName is the hostname a device reported, or its ID if it has none.
func savedDeviceName(saved SavedDevice) string {
	if saved.Name != "" {
		return saved.Name
	}
	return saved.DeviceID
}

// withSavedDeviceName names a discovered device after what it reported when
// last connected, unless reverse DNS already provided a name.
func (a *App) withSavedDeviceName(device discovery.Device) discovery.Device {
	if device.Host != "" {
		return device
	}
	if name := savedDeviceName(a.prefs.Devices[device.BaseURL]); name != "" {
		device.Name = name
	}
	return device
}

// rememberConnectedDevice records what the connected device reports, and when
// it was last connected (once per connection).
func (a *App) rememberConnectedDevice(snap session.Snapshot) {
	baseURL := a.cfg.BaseURL
	if snap.Phase != session.PhaseConnected || baseURL == "" {
		return
	}
	prev := a.prefs.Devices[baseURL]
	next := prev
	if name := strings.TrimSpace(snap.Hostname); name != "" {
		next.Name = name
	}
	if snap.DeviceID != "" {
		next.DeviceID = snap.DeviceID
	}
	if snap.AppVersion != "" {
		next.AppVersion = snap.AppVersion
	}
	if !a.deviceConnectionRecorded {
		next.LastConnected = time.Now()
		a.deviceConnectionRecorded = true
	}
	if next == prev {
		return
	}
	if a.prefs.Devices == nil {
		a.prefs.Devices = map[string]SavedDevice{}
	}
	a.prefs.Devices[baseURL] = next
	a.savePreferences()

	for i := range a.discovered {
		if a.discovered[i].BaseURL == baseURL {
			a.discovered[i] = a.withSavedDeviceName(a.discovered[i])
		}
	}
	a.sortDiscovered()
}

// launcherDeviceDetail is the second line of a discovered device: its address,
// plus the version and last connection remembered from earlier sessions.
func (a *App) launcherDeviceDetail(device discovery.Device) string {
	return deviceDetail(device, a.prefs.Devices[device.BaseURL], time.Now())
}

func deviceDetail(device discovery.Device, saved SavedDevice, now time.Time) string {
	address := device.IP
	if address == "" {
		address = device.BaseURL
	}
	parts := []string{address}
	if saved.Name != "" && saved.DeviceID != "" && saved.Name != saved.DeviceID {
		parts = append(parts, saved.DeviceID)
	}
	if version := saved.AppVersion; version != "" {
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		parts = append(parts, version)
	}
	if !saved.LastConnected.IsZero() {
		parts = append(parts, "last connected "+humanTimeAgo(now.Sub(saved.LastConnected)))
	}
	return strings.Join(parts, " · ")
}

func humanTimeAgo(age time.Duration) string {
	switch {
	case age < time.Minute:
		return "just now"
	case age < time.Hour:
		return fmt.Sprintf("%dm ago", int(age.Minutes()))
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(age.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(age.Hours()/24))
	}
}
