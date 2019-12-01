package utils

import (
	"os"
)

// Get value from env
func GetEnvVal(envName, defaultVal string) string {
	if len(os.Getenv(envName)) > 0 {
		return os.Getenv(envName)
	}

	return defaultVal
}
