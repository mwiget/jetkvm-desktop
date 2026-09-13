//go:build ios

package app

import "errors"

// TODO(ipados): present UIDocumentPickerViewController from the host app.
func chooseDiskImage() (path string, cancelled bool, err error) {
	return "", false, errors.New("choosing local disk images is not supported on iPadOS yet")
}
