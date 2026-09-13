package app

import (
	"testing"

	"github.com/lkarlslund/jetkvm-desktop/pkg/discovery"
	"github.com/lkarlslund/jetkvm-desktop/pkg/session"
)

func TestRememberDeviceNameNamesDiscoveredDevices(t *testing.T) {
	a := &App{cfg: Config{BaseURL: "http://192.168.1.50"}}
	a.addDiscoveredDevice(discovery.Device{Name: "192.168.1.50", BaseURL: "http://192.168.1.50", IP: "192.168.1.50"})

	a.rememberDeviceName(session.Snapshot{Phase: session.PhaseConnecting, Hostname: "rack-kvm"})
	if a.prefs.DeviceNames["http://192.168.1.50"] != "" {
		t.Fatal("name should only be saved once connected")
	}

	a.rememberDeviceName(session.Snapshot{Phase: session.PhaseConnected, Hostname: "rack-kvm"})
	if got := a.prefs.DeviceNames["http://192.168.1.50"]; got != "rack-kvm" {
		t.Fatalf("saved name = %q, want rack-kvm", got)
	}
	if got := a.discovered[0].Name; got != "rack-kvm" {
		t.Fatalf("listed name = %q, want rack-kvm", got)
	}

	if got := loadPreferences().DeviceNames["http://192.168.1.50"]; got != "rack-kvm" {
		t.Fatalf("persisted name = %q, want rack-kvm", got)
	}
}

func TestSavedNameDoesNotOverrideReverseDNS(t *testing.T) {
	a := &App{prefs: Preferences{DeviceNames: map[string]string{"http://192.168.1.50": "rack-kvm"}}}

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
