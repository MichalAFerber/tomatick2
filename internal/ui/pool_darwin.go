//go:build darwin

package ui

/*
#cgo LDFLAGS: -lobjc

// The Objective-C runtime's pool entry points, which @autoreleasepool compiles
// to. They are called directly because a pool opened in Go must stay open
// across a Go callback, and because this package builds with -fobjc-arc
// (dock_darwin.go), where NSAutoreleasePool is unavailable.
void *objc_autoreleasePoolPush(void);
void objc_autoreleasePoolPop(void *context);
*/
import "C"

import "runtime"

// withAutoreleasePool runs fn inside an Objective-C autorelease pool and drains
// the pool when fn returns.
//
// fyne runs fyne.Do callbacks on the main thread between event polls, outside
// the autorelease pool that wraps its event polling. Tray updates made from
// those callbacks can leave autoreleased Cocoa objects that are never released:
// Tomatick 2.0.0 retained a decoded copy of its menu-bar icon on every
// one-second tick, about 28 KB/s (tomatick2#15).
//
// A pool must be popped on the thread that pushed it, so the goroutine stays
// locked to its thread for the call. Locks nest, so this is safe on fyne's
// already-locked main goroutine.
func withAutoreleasePool(fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pool := C.objc_autoreleasePoolPush()
	defer C.objc_autoreleasePoolPop(pool)
	fn()
}
