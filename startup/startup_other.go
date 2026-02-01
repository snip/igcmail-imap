//go:build !windows && !darwin

package startup

// Enabled returns false on unsupported platforms (no run-at-startup support).
func Enabled() (bool, error) {
	return false, nil
}

// SetEnabled is a no-op on unsupported platforms.
func SetEnabled(enabled bool) error {
	return nil
}
