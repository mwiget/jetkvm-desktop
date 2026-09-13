package app

import (
	"testing"
	"time"

	"github.com/lkarlslund/jetkvm-desktop/pkg/discovery"
	"github.com/lkarlslund/jetkvm-desktop/pkg/session"
)

func TestRememberConnectedDeviceNamesDiscoveredDevices(t *testing.T) {
	a := &App{cfg: Config{BaseURL: "http://192.168.1.50"}}
	a.addDiscoveredDevice(discovery.Device{Name: "192.168.1.50", BaseURL: "http://192.168.1.50", IP: "192.168.1.50"})

	a.rememberConnectedDevice(session.Snapshot{Phase: session.PhaseConnecting, Hostname: "rack-kvm"})
	if _, ok := a.prefs.Devices["http://192.168.1.50"]; ok {
		t.Fatal("device should only be recorded once connected")
	}

	a.rememberConnectedDevice(session.Snapshot{Phase: session.PhaseConnected, Hostname: "rack-kvm", DeviceID: "abc123", AppVersion: "0.4.8"})
	saved := a.prefs.Devices["http://192.168.1.50"]
	if saved.Name != "rack-kvm" || saved.DeviceID != "abc123" || saved.AppVersion != "0.4.8" || saved.LastConnected.IsZero() {
		t.Fatalf("saved device = %+v", saved)
	}
	if got := a.discovered[0].Name; got != "rack-kvm" {
		t.Fatalf("listed name = %q, want rack-kvm", got)
	}
	if got := loadPreferences().Devices["http://192.168.1.50"].Name; got != "rack-kvm" {
		t.Fatalf("persisted name = %q, want rack-kvm", got)
	}

	// Later ticks of the same connection must not rewrite the timestamp.
	firstSeen := saved.LastConnected
	a.rememberConnectedDevice(session.Snapshot{Phase: session.PhaseConnected, Hostname: "rack-kvm", DeviceID: "abc123", AppVersion: "0.4.8"})
	if !a.prefs.Devices["http://192.168.1.50"].LastConnected.Equal(firstSeen) {
		t.Fatal("last connected time changed within one connection")
	}
}

func TestSavedDeviceFallsBackToDeviceID(t *testing.T) {
	a := &App{cfg: Config{BaseURL: "http://192.168.1.50"}}
	a.rememberConnectedDevice(session.Snapshot{Phase: session.PhaseConnected, DeviceID: "abc123"})
	a.addDiscoveredDevice(discovery.Device{Name: "192.168.1.50", BaseURL: "http://192.168.1.50"})
	if got := a.discovered[0].Name; got != "abc123" {
		t.Fatalf("listed name = %q, want device ID", got)
	}
}

func TestSavedNameDoesNotOverrideReverseDNS(t *testing.T) {
	a := &App{prefs: Preferences{Devices: map[string]SavedDevice{"http://192.168.1.50": {Name: "rack-kvm"}}}}

	a.addDiscoveredDevice(discovery.Device{Name: "192.168.1.51", BaseURL: "http://192.168.1.51"})
	a.addDiscoveredDevice(discovery.Device{Name: "192.168.1.50", BaseURL: "http://192.168.1.50"})
	a.addDiscoveredDevice(discovery.Device{Name: "kvm.lan", Host: "kvm.lan", BaseURL: "http://192.168.1.52"})

	names := map[string]string{}
	for _, device := range a.discovered {
		names[device.BaseURL] = device.Name
	}
	if names["http://192.168.1.50"] != "rack-kvm" || names["http://192.168.1.51"] != "192.168.1.51" || names["http://192.168.1.52"] != "kvm.lan" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestDeviceDetail(t *testing.T) {
	now := time.Unix(100000, 0)
	device := discovery.Device{IP: "192.168.1.50", BaseURL: "http://192.168.1.50"}

	if got := deviceDetail(device, SavedDevice{}, now); got != "192.168.1.50" {
		t.Fatalf("unknown device detail = %q", got)
	}
	saved := SavedDevice{Name: "rack-kvm", DeviceID: "abc123", AppVersion: "0.4.8", LastConnected: now.Add(-3 * time.Hour)}
	if got, want := deviceDetail(device, saved, now), "192.168.1.50 · abc123 · v0.4.8 · last connected 3h ago"; got != want {
		t.Fatalf("detail = %q, want %q", got, want)
	}
}
