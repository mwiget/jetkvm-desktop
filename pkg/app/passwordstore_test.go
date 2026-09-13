package app

import (
	"testing"

	"github.com/lkarlslund/jetkvm-desktop/pkg/discovery"
)

type fakePasswordStore struct {
	passwords map[string]string
	saves     int
}

func newFakePasswordStore() *fakePasswordStore {
	return &fakePasswordStore{passwords: map[string]string{}}
}

func (f *fakePasswordStore) Load(baseURL string) (string, bool) {
	password, ok := f.passwords[baseURL]
	return password, ok
}

func (f *fakePasswordStore) Save(baseURL, password string) error {
	f.saves++
	f.passwords[baseURL] = password
	return nil
}

func (f *fakePasswordStore) Delete(baseURL string) error {
	delete(f.passwords, baseURL)
	return nil
}

const testDeviceURL = "http://192.168.1.50"

func TestSavedPasswordsSkipIdenticalSavesAndTrackState(t *testing.T) {
	store := newFakePasswordStore()
	saved := newSavedPasswords(store)

	if saved.Has(testDeviceURL) {
		t.Fatal("no password should be saved yet")
	}
	saved.Save(testDeviceURL, "secret")
	saved.Save(testDeviceURL, "secret")
	if store.saves != 1 {
		t.Fatalf("store saves = %d, want 1", store.saves)
	}
	if !saved.Has(testDeviceURL) {
		t.Fatal("password should be reported as saved")
	}
	saved.Save(testDeviceURL, "")
	if saved.Has(testDeviceURL) || len(store.passwords) != 0 {
		t.Fatal("saving an empty password should remove the saved one")
	}
}

func TestPasswordForConnectPrefersEnteredPassword(t *testing.T) {
	store := newFakePasswordStore()
	store.passwords[testDeviceURL] = "saved"
	a := &App{passwords: newSavedPasswords(store)}

	if got := a.passwordForConnect(testDeviceURL); got != "saved" || !a.usingSavedPassword {
		t.Fatalf("got %q (saved=%v), want saved password", got, a.usingSavedPassword)
	}

	a.launcherPassword = "typed"
	if got := a.passwordForConnect(testDeviceURL); got != "typed" || a.usingSavedPassword {
		t.Fatalf("got %q (saved=%v), want typed password", got, a.usingSavedPassword)
	}
}

func TestSuccessfulLoginSavesAndRejectedSavedPasswordIsForgotten(t *testing.T) {
	store := newFakePasswordStore()
	a := &App{passwords: newSavedPasswords(store), cfg: Config{BaseURL: testDeviceURL, Password: "secret"}}

	a.rememberDevicePassword()
	if store.passwords[testDeviceURL] != "secret" {
		t.Fatal("password should be saved after a successful login")
	}

	// A typed password that fails must not discard the saved one.
	a.usingSavedPassword = false
	a.forgetRejectedSavedPassword()
	if store.passwords[testDeviceURL] != "secret" {
		t.Fatal("saved password removed after a typed password failed")
	}

	a.usingSavedPassword = true
	a.forgetRejectedSavedPassword()
	if _, ok := store.passwords[testDeviceURL]; ok || a.cfg.Password != "" || a.usingSavedPassword {
		t.Fatal("rejected saved password should be forgotten")
	}
}

func TestNilSavedPasswordsIsSafe(t *testing.T) {
	a := &App{cfg: Config{BaseURL: testDeviceURL, Password: "secret"}}
	a.rememberDevicePassword()
	if got := a.passwordForConnect(testDeviceURL); got != "secret" {
		t.Fatalf("got %q, want configured password", got)
	}
	if a.passwords.Has(testDeviceURL) {
		t.Fatal("nil store should never report saved passwords")
	}
}

func TestLauncherDeviceState(t *testing.T) {
	if got := launcherDeviceState(discovery.Device{IsSetup: false}, true); got != "Needs setup" {
		t.Fatalf("got %q", got)
	}
	if got := launcherDeviceState(discovery.Device{IsSetup: true}, true); got != "Password saved" {
		t.Fatalf("got %q", got)
	}
	if got := launcherDeviceState(discovery.Device{IsSetup: true}, false); got != "" {
		t.Fatalf("got %q", got)
	}
}
