//go:build !ios

package app

// Desktop builds do not save device passwords. On macOS, Keychain items are
// tied to the binary that created them, so every rebuild of an unsigned
// desktop build would prompt for Keychain access.
type noPasswordStore struct{}

func platformPasswordStore() devicePasswordStore {
	return noPasswordStore{}
}

func (noPasswordStore) Load(string) (string, bool) { return "", false }

func (noPasswordStore) Save(string, string) error { return nil }

func (noPasswordStore) Delete(string) error { return nil }
