//go:build linux && !android

package app

import (
	"os"
	"testing"
)

// TestSecretServiceRoundTrip needs a running Secret Service, so it runs only
// with JETKVM_TEST_SECRET_SERVICE=1.
func TestSecretServiceRoundTrip(t *testing.T) {
	if os.Getenv("JETKVM_TEST_SECRET_SERVICE") == "" {
		t.Skip("set JETKVM_TEST_SECRET_SERVICE=1 to test against the desktop keyring")
	}
	const baseURL = "http://jetkvm-desktop-test.invalid"
	store := platformPasswordStore()
	t.Cleanup(func() { _ = store.Delete(baseURL) })

	if err := store.Save(baseURL, "s3cret"); err != nil {
		t.Fatal(err)
	}
	if got, ok := store.Load(baseURL); !ok || got != "s3cret" {
		t.Fatalf("Load = %q, %v", got, ok)
	}
	if err := store.Delete(baseURL); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Load(baseURL); ok {
		t.Fatal("password still there after Delete")
	}
	if err := store.Delete(baseURL); err != nil {
		t.Fatalf("deleting a missing password: %v", err)
	}
}
