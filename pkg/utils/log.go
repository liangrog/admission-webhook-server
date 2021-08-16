package utils

import (
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Set zero log level
func GetZeroLogLevel(level string) zerolog.Level {
	var zl zerolog.Level
	switch strings.ToLower(level) {
	case "panic":
		zl = zerolog.PanicLevel
	case "fatal":
		zl = zerolog.FatalLevel
	case "error":
		zl = zerolog.ErrorLevel
	case "warn":
		zl = zerolog.WarnLevel
	case "debug":
		zl = zerolog.DebugLevel
	case "info":
		zl = zerolog.InfoLevel
	case "trace":
		zl = zerolog.TraceLevel
	default:
		zl = zerolog.InfoLevel
	}

	return zl
}

const (
	// Shared const for logger key
	LOGGER_LOCATION_KEY = "location"
)

// Create a logger with location string
func GetLogger(location string) zerolog.Logger {
	return log.With().Str(LOGGER_LOCATION_KEY, location).Logger()
}
