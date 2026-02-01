//go:build windows

package startup

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const appName = "igcMailImap"

// Enabled returns whether "run at startup" is currently enabled (registry).
func Enabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()
	_, _, err = k.GetStringValue(appName)
	if err == registry.ErrNotExist {
		return false, nil
	}
	return err == nil, err
}

// SetEnabled enables or disables running at Windows startup.
func SetEnabled(enabled bool) error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	// Use quoted path in case of spaces
	cmd := `"` + path + `"`
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if enabled {
		return k.SetStringValue(appName, cmd)
	}
	return k.DeleteValue(appName)
}
