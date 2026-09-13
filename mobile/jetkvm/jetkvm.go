// Package jetkvm is the gomobile binding used by the iPadOS host app in ios/.
//
// Build it with scripts/build-ios, which runs `ebitenmobile bind` and produces
// ios/build/Jetkvm.xcframework exposing JetkvmEbitenViewController.
package jetkvm

import (
	"context"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/lkarlslund/jetkvm-desktop/pkg/app"
	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
)

func init() {
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

// Dummy exists so gobind has an exported symbol to generate bindings for.
func Dummy() {}
