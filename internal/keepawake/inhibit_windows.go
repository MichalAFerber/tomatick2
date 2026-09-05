//go:build windows

package keepawake

import "golang.org/x/sys/windows"

const (
	esContinuous      = 0x80000000
	esSystemRequired  = 0x00000001
	esDisplayRequired = 0x00000002
)

func inhibit() (func(), error) {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := kernel32.NewProc("SetThreadExecutionState")
	r1, _, err := proc.Call(uintptr(esContinuous | esSystemRequired | esDisplayRequired))
	if r1 == 0 {
		return nil, err
	}
	return func() {
		_, _, _ = proc.Call(uintptr(esContinuous))
	}, nil
}
