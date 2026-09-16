// Package jetkvm is the gomobile binding used by the iPadOS host app in ios/.
//
// Build it with scripts/build-ios, which runs `ebitenmobile bind` and produces
// ios/build/Jetkvm.xcframework exposing JetkvmEbitenViewController.
package jetkvm

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/lkarlslund/jetkvm-desktop/pkg/app"
	"github.com/lkarlslund/jetkvm-desktop/pkg/hostinput"
	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
)

func init() {
	// iOS discards stderr unless launched with `simctl launch --console`. Keep
	// logs and Go panics in the app's temporary directory instead, where they
	// can be pulled from the app container.
	logDirs := []string{os.TempDir()}
	// On a Mac, TMPDIR is the user's shared temporary directory, which the
	// sandbox doesn't let the app write; the container is CFFIXED_USER_HOME.
	if home := os.Getenv("CFFIXED_USER_HOME"); home != "" {
		logDirs = append(logDirs, filepath.Join(home, "tmp"))
	}
	for _, dir := range logDirs {
		if logFile, err := os.Create(filepath.Join(dir, "jetkvm.log")); err == nil {
			_ = syscall.Dup2(int(logFile.Fd()), 2)
			break
		}
	}

	// There is no command line on iPadOS. JETKVM_URL, JETKVM_PASSWORD and
	// JETKVM_DESKTOP_LOG_LEVEL can be set from the Xcode scheme, or with
	// SIMCTL_CHILD_-prefixed variables for `xcrun simctl launch`.
	_ = logging.Configure(os.Getenv("JETKVM_DESKTOP_LOG_LEVEL"))

	clientApp, err := app.New(app.Config{
		BaseURL:    os.Getenv("JETKVM_URL"),
		Password:   os.Getenv("JETKVM_PASSWORD"),
		RPCTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	clientApp.Start(context.Background())

	ebiten.SetTPS(ebiten.SyncWithFPS)
	mobile.SetGame(clientApp)
}

// The functions below are called from the Swift host on the main thread.
// Button masks: bit 0 primary, bit 1 secondary, bit 2 middle, bit 3 back, bit 4 forward.

// PointerMoved reports an absolute pointer position in view points.
func PointerMoved(x, y float64) { app.HostPointerMoved(x, y) }

// PointerMovedBy reports relative motion while the pointer is locked.
func PointerMovedBy(dx, dy float64) { app.HostPointerMovedBy(dx, dy) }

// PointerButtons reports the full set of currently pressed buttons.
func PointerButtons(buttons int) { app.HostPointerButtons(buttons) }

// PointerScrolled reports wheel motion in wheel units (positive y scrolls up).
func PointerScrolled(dx, dy float64) { app.HostPointerScrolled(dx, dy) }

// PointerLockRequested reports whether relative mouse mode wants the pointer locked.
func PointerLockRequested() bool { return app.HostPointerLockRequested() }

// PointerHidden reports whether the pointer should be hidden over the view.
func PointerHidden() bool { return app.HostPointerHidden() }

// SetDarkMode reports the system appearance.
func SetDarkMode(dark bool) { app.HostSetDarkMode(dark) }

// Editing keys for KeyTap.
const (
	KeyBackspace = 1
	KeyEnter     = 2
	KeyTab       = 3
	KeyEscape    = 4
	KeyLeft      = 5
	KeyRight     = 6
	KeyDelete    = 7
)

var editingKeys = map[int]ebiten.Key{
	KeyBackspace: ebiten.KeyBackspace,
	KeyEnter:     ebiten.KeyEnter,
	KeyTab:       ebiten.KeyTab,
	KeyEscape:    ebiten.KeyEscape,
	KeyLeft:      ebiten.KeyLeft,
	KeyRight:     ebiten.KeyRight,
	KeyDelete:    ebiten.KeyDelete,
}

// InsertText delivers text typed on the on-screen keyboard.
func InsertText(text string) { hostinput.InsertText(text) }

// KeyTap delivers an editing key from the on-screen keyboard.
func KeyTap(key int) {
	if ebitenKey, ok := editingKeys[key]; ok {
		hostinput.TapKey(ebitenKey)
	}
}

// TextInputActive reports whether the on-screen keyboard should be shown.
func TextInputActive() bool { return app.HostTextInputActive() }

// TextInputDismissed reports that the user hid the on-screen keyboard.
func TextInputDismissed() { app.HostTextInputDismissed() }

// AppWillResignActive reports that the scene stopped being the active one.
func AppWillResignActive() { app.HostAppWillResignActive() }

// AppDidEnterBackground reports that the scene moved to the background.
func AppDidEnterBackground() { app.HostAppDidEnterBackground() }

// AppDidBecomeActive reports that the scene became active again.
func AppDidBecomeActive() { app.HostAppDidBecomeActive() }

// FilePickerRequested reports, once per request, that a disk image should be chosen.
func FilePickerRequested() bool { return app.HostFilePickerRequested() }

// FilePicked delivers the local path of the chosen disk image.
func FilePicked(path string) { app.HostFilePicked(path) }

// OpenURL delivers a jetkvm://host[:port] URL to connect to.
func OpenURL(url string) { app.HostOpenURL(url) }
