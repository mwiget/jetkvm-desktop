// Copyright 2019 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metal

// Suppress the warnings about availability guard with -Wno-unguarded-availability-new.
// It is because old Xcode (8 or older?) does not accept @available syntax.

// #cgo CFLAGS: -Wno-unguarded-availability-new -x objective-c
// #cgo LDFLAGS: -framework UIKit -framework QuartzCore -framework Foundation -framework CoreGraphics
//
// #import <UIKit/UIKit.h>
// #import <stdatomic.h>
//
// #cgo noescape addSublayer
// #cgo nocallback addSublayer
// static void addSublayer(void* view, void* sublayer) {
//   CALayer* layer = ((UIView*)view).layer;
//   [layer addSublayer:(CALayer*)sublayer];
// }
//
// #cgo noescape setFrame
// #cgo nocallback setFrame
// static void setFrameOnMainThread(void* cametal, void* uiview) {
//   CGSize size = ((UIView*)uiview).frame.size;
//   ((CALayer*)cametal).frame = CGRectMake(0, 0, size.width, size.height);
// }
//
// // JetKVM: this runs on the render thread inside its CADisplayLink callback.
// // Waiting for the main thread here deadlocks when UIKit, on the main thread,
// // moves the display links to another display as the scene comes back to the
// // foreground on iPadOS 27, since that waits for the callback to finish. Hand
// // the update to the main thread instead of waiting for it.
// static atomic_bool setFramePending;
// static void setFrame(void* cametal, void* uiview) {
//   if ([NSThread isMainThread]) {
//     setFrameOnMainThread(cametal, uiview);
//     return;
//   }
//   if (atomic_exchange(&setFramePending, true)) {
//     return;
//   }
//   dispatch_async(dispatch_get_main_queue(), ^{
//     atomic_store(&setFramePending, false);
//     setFrameOnMainThread(cametal, uiview);
//   });
// }
import "C"

import (
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

func (v *view) setWindow(window uintptr) {
	panic("metal: setWindow is not available on iOS")
}

func (v *view) setUIView(uiview uintptr) {
	v.uiview = uiview
}

func (v *view) update() {
	v.once.Do(func() {
		if v.ml.Layer() == nil {
			panic("metal: CAMetalLayer is not initialized yet")
		}
		C.addSublayer(unsafe.Pointer(v.uiview), v.ml.Layer())
	})
	C.setFrame(v.ml.Layer(), unsafe.Pointer(v.uiview))
}

const (
	storageMode         = mtl.StorageModeShared
	resourceStorageMode = mtl.ResourceStorageModeShared
)

func (v *view) nextDrawable() ca.MetalDrawable {
	d, err := v.ml.NextDrawable()
	if err != nil {
		// Drawable is nil. This can happen at the initial state. Let's wait and see.
		return ca.MetalDrawable{}
	}
	return d
}

func (v *view) finishDrawableUsage() {
	// Do nothing.
}

func (v *view) initializeOS() error {
	// Do nothing.
	return nil
}
