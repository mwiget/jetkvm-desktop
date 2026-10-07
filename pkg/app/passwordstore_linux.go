//go:build linux && !android

package app

import (
	"errors"
	"fmt"
	"time"

	"github.com/zalando/go-keyring"
)

// secretService names this app's device passwords in the desktop's Secret
// Service (GNOME Keyring, KWallet), matching the Keychain service on iPadOS.
const secretService = "io.github.mwiget.jetkvm.device-password"

// secretServiceTimeout bounds a Secret Service call. A locked keyring asks the
// user to unlock it, and the launcher must not freeze while it waits.
const secretServiceTimeout = 3 * time.Second

type secretServicePasswordStore struct{}

func platformPasswordStore() devicePasswordStore {
	return secretServicePasswordStore{}
}

// withTimeout runs a Secret Service call, giving up on it after
// secretServiceTimeout. The call itself carries on, and its result is dropped.
func withTimeout[T any](call func() (T, error)) (T, error) {
	type result struct {
		value T
		err   error
	}
	done := make(chan result, 1)
	go func() {
		value, err := call()
		done <- result{value, err}
	}()
	select {
	case r := <-done:
		return r.value, r.err
	case <-time.After(secretServiceTimeout):
		var zero T
		return zero, errors.New("secret service: no answer")
	}
}

func (secretServicePasswordStore) Load(baseURL string) (string, bool) {
	password, err := withTimeout(func() (string, error) {
		return keyring.Get(secretService, baseURL)
	})
	if err != nil {
		return "", false
	}
	return password, true
}

func (secretServicePasswordStore) Save(baseURL, password string) error {
	_, err := withTimeout(func() (struct{}, error) {
		return struct{}{}, keyring.Set(secretService, baseURL, password)
	})
	if err != nil {
		return fmt.Errorf("secret service: save: %w", err)
	}
	return nil
}

func (secretServicePasswordStore) Delete(baseURL string) error {
	_, err := withTimeout(func() (struct{}, error) {
		return struct{}{}, keyring.Delete(secretService, baseURL)
	})
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("secret service: delete: %w", err)
	}
	return nil
}
