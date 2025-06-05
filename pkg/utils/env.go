package utils

import (
	"os"
	"runtime"
	"strings"
)

// GetEnvironmentVariableCaseInsensitive gets environment variable in a case-insensitive manner
// This is particularly useful for Windows where environment variables are case-insensitive
func GetEnvironmentVariableCaseInsensitive(key string) string {
	// First try exact match
	if value := os.Getenv(key); value != "" {
		return value
	}

	// On Windows, try case-insensitive search
	if runtime.GOOS == "windows" {
		key = strings.ToUpper(key)
		for _, env := range os.Environ() {
			pair := strings.SplitN(env, "=", 2)
			if len(pair) == 2 && strings.ToUpper(pair[0]) == key {
				return pair[1]
			}
		}
	}

	return ""
}
