package app

import (
	"fmt"
	"os"
	"testing"
)

// TestMain keeps tests from reading or overwriting the real preferences file:
// App.savePreferences writes to os.UserConfigDir, which derives from these
// variables on macOS, Linux and Windows. Saved passwords likewise go to a fake
// store rather than the desktop's keyring.
func TestMain(m *testing.M) {
	newPasswordStore = func() devicePasswordStore { return newFakePasswordStore() }
	dir, err := os.MkdirTemp("", "jetkvm-app-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, name := range []string{"HOME", "XDG_CONFIG_HOME", "AppData"} {
		if err := os.Setenv(name, dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
