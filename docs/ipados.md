# iPadOS app

The iPadOS app runs the same Go client as the desktop build. `ebitenmobile bind`
compiles `./mobile/jetkvm` into `ios/build/Jetkvm.xcframework`, and a small UIKit
host app in `ios/` (generated with XcodeGen) embeds it.

## Requirements

- Xcode with the iOS SDK (developed against Xcode 27 beta), Go matching `go.mod`
- XcodeGen: `brew install xcodegen`
- `ebitenmobile` is installed automatically by the build script at the Ebitengine
  version from `go.mod`

## Build

```bash
scripts/build-ios
open ios/JetKVM.xcodeproj
```

The script builds the framework for device and simulator and regenerates
`ios/JetKVM.xcodeproj` from `ios/project.yml`. Both outputs are ignored by git.
Re-run it after changing Go code; Swift-only changes just need a rebuild in Xcode.

To run on a device, select your team under Signing & Capabilities (or change
`DEVELOPMENT_TEAM` in `ios/project.yml`).

The generated scheme runs the app without the debugger. Go's runtime signals
itself constantly to preempt goroutines, and LLDB stops on each signal, so with
the debugger attached the app lags badly and misses key presses. Attach with
Debug > Attach to Process only when you need breakpoints.

## Mac

The same iPad build runs on Apple Silicon Macs as a "Designed for iPad" app; no
separate target or code is needed. Mac Catalyst is not an option because
Ebitengine has no Catalyst backend (`gomobile`'s Catalyst target builds with
`GOOS=darwin`, which drops the UIKit code).

```bash
scripts/build-mac --install
```

The script runs `scripts/build-ios`, archives the app, exports a
development-signed `ios/build/export/JetKVM.ipa` and, with `--install`, opens it,
which installs `/Applications/JetKVM.app`. The Mac must be in the team's
provisioning profile; the first build registers it. From Xcode, pick the
"My Mac (Designed for iPad)" run destination instead.

A copied `.app` with a hand-made `Wrapper/` directory is rejected by Gatekeeper
("bad app wrapper"); install through the `.ipa`. Opening an `.ipa` while the app
is already installed adds `JetKVM 2.app` instead of upgrading it, so `--install`
first quits the app and moves existing copies to the Trash (Finder may ask for
an administrator password, since the installed wrapper is owned by root).

On the Mac the Go log is `tmp/jetkvm.log` in the app's container under
`~/Library/Containers/<UUID>/Data` (the container is named by a UUID, not the
bundle identifier).

macOS backgrounds the scene as soon as another app comes to the front, even
though the window stays visible and, unlike iPadOS, the process keeps running
with its WebRTC session up. Reconnecting on the way back would throw away a
live session, so `SceneDelegate` reports the background to the Go app only when
`ProcessInfo.processInfo.isiOSAppOnMac` is false; a connection that really is
lost is still picked up by the session controller's own reconnect.

Resigning active does stop the game loop, and with it `video.SetPaused` stops
copying decoded frames into Go memory, which is most of what an idle background
app was costing. VideoToolbox keeps decoding, because its reference frames have
to stay current for the picture to be right afterwards, and it holds the last
one: becoming active again publishes that through `Controller.RefreshVideo`, so
a screen that changed and then went static is on screen immediately instead of
waiting for the device to send something new.

## Running

- The launcher discovers JetKVM devices on the local network. iPadOS asks for
  Local Network access on first launch.
- `jetkvm://host[:port]` opens the app connected to that device, e.g.
  `jetkvm://192.168.1.50`. Use it from Shortcuts or a home screen bookmark.
- Environment variables set in the Xcode scheme (or with `SIMCTL_CHILD_` prefixes
  for `xcrun simctl launch`): `JETKVM_URL`, `JETKVM_PASSWORD`,
  `JETKVM_DESKTOP_LOG_LEVEL`.
- Go logs and panics are written to `tmp/jetkvm.log` inside the app's data
  container (`xcrun simctl get_app_container <device> io.github.mwiget.jetkvm data`
  in the simulator; `xcrun devicectl device copy from --domain-type appDataContainer
  --domain-identifier io.github.mwiget.jetkvm --source tmp/jetkvm.log` on a device).
  At `debug` level the log includes video decode statistics and, every five
  seconds, frame rate, update/draw/upload timings and key send times.

To try the app without hardware, run the emulator on the Mac and point the
simulator at it:

```bash
go run ./test/emulator-serve serve --listen 127.0.0.1:8080
SIMCTL_CHILD_JETKVM_URL=http://127.0.0.1:8080 xcrun simctl launch booted io.github.mwiget.jetkvm
```

## How input and video work

- **Pointer:** Ebitengine only handles touches and keys on iOS. The host
  (`GameViewController.swift`) forwards hover, trackpad/mouse clicks, scrolling
  and, while relative mouse mode locks the pointer, `GCMouse` motion into
  `pkg/app/hostinput.go`. Desktop builds keep using Ebitengine directly.
- **Keyboard:** hardware keyboards go through Ebitengine. Without one, tapping a
  text field (or opening the paste or serial console overlays) shows the on-screen
  keyboard; typing reaches Go through `pkg/hostinput`. The game view stays above
  the keyboard (`HostViewController.swift`).
- **Video:** H.264 is decoded with VideoToolbox (`pkg/video/codec_videotoolbox.go`),
  using the hardware decoder on devices.
- **Virtual media:** choosing a local disk image opens the Files picker; the file
  is copied into the app sandbox and uploaded from there.
- **Background:** iPadOS suspends the WebRTC session; returning to the app after a
  few seconds in the background reconnects.

## Host integration rules

- Do not use GameController (`GCMouse`, `GCKeyboard`) or add the scroll pan
  recognizer before Ebitengine's game view exists. Doing so in `viewDidLoad` kept
  Ebitengine's render thread from starting, leaving a black screen.
- Do not add subviews to `GameViewController.view`; it detects the game view by
  its first subview. Put host views in `HostViewController`.
- The VideoToolbox decoder recreates its session whenever SPS/PPS change. Swapping
  the format into a live session made the simulator's decoder reject every
  following frame.
- The app target's module is `JetKVMApp`; `JetKVM` would clash with the Go
  framework's `Jetkvm` module on case-insensitive file systems.

## Status

Verified in the iPad simulator: launcher and discovery, live video from the
emulator, `jetkvm://` URLs while the app is running, app icon.

Not yet verified on a device: on-screen keyboard (the simulator reports a
hardware keyboard), trackpad and mouse input including pointer lock, virtual
media upload, background reconnect, and hardware video decoding performance.
