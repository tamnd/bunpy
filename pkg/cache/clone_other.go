//go:build !darwin

package cache

import (
	"io"
	"os"
)

// cloneOrCopy copies src to dst using a hardlink on Linux (same filesystem),
// falling back to a plain byte copy if the link fails (cross-device, read-only, etc.).
func cloneOrCopy(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}
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
