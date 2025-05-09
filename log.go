package h2go

import (
	"log"
	"log/slog"
	"os"
)

// respect env variable H2GO_LOG_LEVEL or LOG_LEVEL
var logLevel = "info"

func init() {
	if v := os.Getenv("H2GO_LOG_LEVEL"); v != "" {
		logLevel = v
	} else if v := os.Getenv("LOG_LEVEL"); v != "" {
		logLevel = v
	}
}

// DefaultLogger returns a new logger with the default log level
// set to the value of the H2GO_LOG_LEVEL or LOG_LEVEL environment variable.
func DefaultLogger() *slog.Logger {
	// parse log level
	var level slog.Level
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		level = slog.LevelInfo
	}
	log.Println("log level:", level)
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{AddSource: true, Level: level}))
}
