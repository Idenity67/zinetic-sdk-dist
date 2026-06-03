//go:build !linux && !darwin

package credential

func mlockBytes(_ []byte) bool {
	return false
}

func munlockBytes(_ []byte) {}
