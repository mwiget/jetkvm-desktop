//go:build !ios

package app

import "github.com/sqweek/dialog"

func chooseDiskImage() (path string, cancelled bool, err error) {
	path, err = dialog.File().
		Title("Choose disk image").
		Filter("Disk images", "iso", "img").
		Load()
	if err == dialog.ErrCancelled {
		return "", true, nil
	}
	return path, false, err
}
