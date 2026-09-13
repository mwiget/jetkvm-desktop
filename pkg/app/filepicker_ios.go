//go:build ios

package app

// chooseDiskImage asks the host app to present the document picker. The chosen
// file arrives later through HostFilePicked, so report the synchronous call as
// cancelled.
func chooseDiskImage() (path string, cancelled bool, err error) {
	hostFilePickerRequested.Store(true)
	return "", true, nil
}
