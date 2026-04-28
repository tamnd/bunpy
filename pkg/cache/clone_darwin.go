//go:build darwin

package cache

import (
	"io"
	"os"
	"syscall"
	"unsafe"
)

// SYS_CLONEFILEAT is the macOS clonefile syscall number.
const SYS_CLONEFILEAT = 488

// cloneOrCopy copies src to dst using APFS copy-on-write clone on macOS,
// falling back to a plain byte copy on cross-filesystem or non-APFS paths.
func cloneOrCopy(src, dst string) error {
	srcp, err := syscall.BytePtrFromString(src)
	if err != nil {
		return err
	}
	dstp, err := syscall.BytePtrFromString(dst)
	if err != nil {
		return err
	}
	// clonefileat(AT_FDCWD=-2, src, AT_FDCWD=-2, dst, 0)
	const atFDCWD = ^uintptr(1) // -2 in two's complement
	_, _, errno := syscall.Syscall6(
		SYS_CLONEFILEAT,
		atFDCWD,
		uintptr(unsafe.Pointer(srcp)),
		atFDCWD,
		uintptr(unsafe.Pointer(dstp)),
		0, 0,
	)
	if errno == 0 {
		return nil
	}
	// Fall back to plain copy (cross-device, non-APFS, etc.)
	return plainCopy(src, dst)
}

func plainCopy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}
