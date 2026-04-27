//go:build !windows

package bunpy

import (
	"os"
	"syscall"
	"unsafe"
)

func termColumns() int {
	ws := struct{ Row, Col, Xpixel, Ypixel uint16 }{}
	if _, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	); errno == 0 {
		return int(ws.Col)
	}
	return 80
}

func termRows() int {
	ws := struct{ Row, Col, Xpixel, Ypixel uint16 }{}
	if _, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	); errno == 0 {
		return int(ws.Row)
	}
	return 24
}

func isTerminal() bool {
	ws := struct{ Row, Col, Xpixel, Ypixel uint16 }{}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(os.Stdout.Fd()),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	return errno == 0
}
