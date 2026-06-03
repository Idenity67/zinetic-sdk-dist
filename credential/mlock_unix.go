//go:build linux || darwin

package credential

import "golang.org/x/sys/unix"

func mlockBytes(buf []byte) bool {
	if len(buf) == 0 {
		return false
	}
	err := unix.Mlock(buf)
	return err == nil
}

func munlockBytes(buf []byte) {
	if len(buf) == 0 {
		return
	}
	_ = unix.Munlock(buf)
}
