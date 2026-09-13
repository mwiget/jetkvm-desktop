package app

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
)

// Native shells cannot block for a file dialog, and hand over URLs opened by
// the system. These requests are queued here and applied at the start of a
// tick. Desktop builds never call these.

var (
	hostFilePickerRequested atomic.Bool
	hostRequests            hostRequestQueue
)

// HostFilePickerRequested reports, once per request, that the app wants the
// user to choose a disk image.
func HostFilePickerRequested() bool {
	return hostFilePickerRequested.CompareAndSwap(true, false)
}

// HostFilePicked delivers the local path of the disk image the user chose.
func HostFilePicked(path string) {
	hostRequests.addFile(path)
}

// HostOpenURL delivers a jetkvm://host[:port] URL to connect to.
func HostOpenURL(raw string) {
	hostRequests.addURL(raw)
}

type hostRequestQueue struct {
	mu    sync.Mutex
	files []string
	urls  []string
}

func (q *hostRequestQueue) addFile(path string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.files = append(q.files, path)
}

func (q *hostRequestQueue) addURL(raw string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.urls = append(q.urls, raw)
}

func (q *hostRequestQueue) take() (files, urls []string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	files, urls = q.files, q.urls
	q.files, q.urls = nil, nil
	return files, urls
}

func (a *App) syncHostRequests() {
	files, urls := hostRequests.take()
	for _, path := range files {
		if a.mediaOpen {
			a.applyPickedUploadFile(path)
		}
	}
	for _, raw := range urls {
		target, err := hostURLTarget(raw)
		if err != nil {
			log := logging.Subsystem("app")
			log.Warn().Err(err).Str("url", raw).Msg("ignoring URL")
			continue
		}
		a.connectFromLauncher(target)
	}
}

// hostURLTarget turns jetkvm://host[:port] into a connect target.
func hostURLTarget(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(u.Scheme, "jetkvm") {
		return "", fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("URL has no host")
	}
	return u.Host, nil
}
