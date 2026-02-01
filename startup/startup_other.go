//go:build !windows && !darwin

package startup

// Enabled returns false on non-Windows (no run-at-startup support).
func Enabled() (bool, error) {
	return false, nil
}

// SetEnabled is a no-op on non-Windows.
func SetEnabled(enabled bool) error {
	return nil
}
