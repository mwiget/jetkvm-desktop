package app

import (
	"sync"

	"github.com/lkarlslund/jetkvm-desktop/pkg/discovery"
	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
)

// devicePasswordStore persists device passwords keyed by base URL. iPadOS uses
// the Keychain (passwordstore_ios.go); other platforms do not save passwords.
type devicePasswordStore interface {
	Load(baseURL string) (string, bool)
	Save(baseURL, password string) error
	Delete(baseURL string) error
}

// savedPasswords wraps the platform store and caches which devices have a
// saved password, so the launcher can show it without a lookup every frame.
// Settings actions update passwords from other goroutines, hence the mutex.
type savedPasswords struct {
	store devicePasswordStore
	mu    sync.Mutex
	known map[string]bool
}

func newSavedPasswords(store devicePasswordStore) *savedPasswords {
	return &savedPasswords{store: store, known: map[string]bool{}}
}

func (s *savedPasswords) setKnown(baseURL string, saved bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.known[baseURL] = saved
}

// Load returns the saved password for a device.
func (s *savedPasswords) Load(baseURL string) (string, bool) {
	if s == nil || baseURL == "" {
		return "", false
	}
	password, ok := s.store.Load(baseURL)
	s.setKnown(baseURL, ok)
	return password, ok
}

// Has reports whether a device has a saved password, from cache when possible.
func (s *savedPasswords) Has(baseURL string) bool {
	if s == nil || baseURL == "" {
		return false
	}
	s.mu.Lock()
	saved, cached := s.known[baseURL]
	s.mu.Unlock()
	if cached {
		return saved
	}
	_, ok := s.Load(baseURL)
	return ok
}

// Save stores a device password; an empty password removes the saved one.
func (s *savedPasswords) Save(baseURL, password string) {
	if s == nil || baseURL == "" {
		return
	}
	if password == "" {
		s.Delete(baseURL)
		return
	}
	if existing, ok := s.store.Load(baseURL); ok && existing == password {
		s.setKnown(baseURL, true)
		return
	}
	if err := s.store.Save(baseURL, password); err != nil {
		log := logging.Subsystem("app")
		log.Warn().Err(err).Str("base_url", baseURL).Msg("failed to save device password")
		s.setKnown(baseURL, false)
		return
	}
	s.setKnown(baseURL, true)
}

// Delete removes a device's saved password.
func (s *savedPasswords) Delete(baseURL string) {
	if s == nil || baseURL == "" {
		return
	}
	if err := s.store.Delete(baseURL); err != nil {
		log := logging.Subsystem("app")
		log.Warn().Err(err).Str("base_url", baseURL).Msg("failed to delete device password")
	}
	s.setKnown(baseURL, false)
}

// passwordForConnect picks the password for a new connection: one entered or
// configured for this session first, otherwise the device's saved password.
func (a *App) passwordForConnect(baseURL string) string {
	a.usingSavedPassword = false
	if password := a.effectivePassword(); password != "" {
		return password
	}
	if saved, ok := a.passwords.Load(baseURL); ok {
		a.usingSavedPassword = true
		return saved
	}
	return ""
}

// rememberDevicePassword saves the password of a connection that just succeeded.
func (a *App) rememberDevicePassword() {
	if a.cfg.Password == "" {
		return
	}
	a.passwords.Save(a.cfg.BaseURL, a.cfg.Password)
}

// forgetRejectedSavedPassword drops a saved password the device rejected, so
// the prompt asks for the current one.
func (a *App) forgetRejectedSavedPassword() {
	if !a.usingSavedPassword {
		return
	}
	a.passwords.Delete(a.cfg.BaseURL)
	a.usingSavedPassword = false
	a.cfg.Password = ""
}

// launcherDeviceState is the status shown on the right of a discovered device.
func launcherDeviceState(device discovery.Device, passwordSaved bool) string {
	switch {
	case !device.IsSetup:
		return "Needs setup"
	case passwordSaved:
		return "Password saved"
	default:
		return ""
	}
}
