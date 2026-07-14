//go:build windows

package progress

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	setConsoleMode     = kernel32.NewProc("SetConsoleMode")
	getConsoleMode     = kernel32.NewProc("GetConsoleMode")
	getStdHandle       = kernel32.NewProc("GetStdHandle")
)

const enableVirtualTerminalProcessing = 0x0004

func vtEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	handle, _, _ := getStdHandle.Call(uintptr(0xFFFFFFF5)) // STD_OUTPUT_HANDLE (-11)
	if handle == 0 || handle == uintptr(syscall.InvalidHandle) {
		return false
	}
	var mode uint32
	ret, _, _ := getConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode)))
	if ret == 0 {
		return false
	}
	mode |= enableVirtualTerminalProcessing
	ret, _, _ = setConsoleMode.Call(handle, uintptr(mode))
	return ret != 0
}
