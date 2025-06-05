package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetUserConfigDir returns the user configuration directory path
func GetUserConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".llm-caller"), nil
}

// GetFilePermissions returns appropriate file permissions for the current platform
func GetFilePermissions() os.FileMode {
	if runtime.GOOS == "windows" {
		return 0666
	}
	return 0644
}

// GetDirPermissions returns appropriate directory permissions for the current platform
func GetDirPermissions() os.FileMode {
	if runtime.GOOS == "windows" {
		return 0777
	}
	return 0755
}

// CreateDirWithPlatformPermissions creates a directory with appropriate permissions for the platform
func CreateDirWithPlatformPermissions(dirname string) error {
	return os.MkdirAll(dirname, GetDirPermissions())
}
