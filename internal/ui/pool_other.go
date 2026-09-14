//go:build !darwin

package ui

// withAutoreleasePool is a no-op outside macOS; see pool_darwin.go.
func withAutoreleasePool(fn func()) { fn() }
