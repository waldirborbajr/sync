package logger

import (
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	once     sync.Once
	instance zerolog.Logger
)

// InitLogger initializes the logger with configurations for console and file output
func InitLogger(debug bool) zerolog.Logger {
	once.Do(func() {
		// Configure console output
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			FormatLevel: func(i interface{}) string {
				return ""
			},
			FormatMessage: func(i interface{}) string {
				return ""
			},
		}

		// Configure file output
		file, err := os.OpenFile("sync.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Fallback to console only if file cannot be opened
			instance = zerolog.New(consoleWriter).
				Level(zerolog.InfoLevel).
				With().
				Timestamp().
				Logger()
			instance.Error().Err(err).Msg("Failed to open sync.log, logging to console only")
			return
		}

		// Set up multi-writer for both console and file
		multiWriter := zerolog.MultiLevelWriter(consoleWriter, file)

		// Set log level and additional fields based on debug mode
		level := zerolog.InfoLevel
		var logger zerolog.Logger
		if debug {
			level = zerolog.DebugLevel
			// Add caller and stack trace for detailed troubleshooting
			logger = zerolog.New(multiWriter).
				Level(level).
				With().
				Timestamp().
				Caller(). // Include file and line number
				Stack().  // Include stack trace for errors
				Logger()
		} else {
			logger = zerolog.New(multiWriter).
				Level(level).
				With().
				Timestamp().
				Logger()
		}

		instance = logger
	})

	// Log initialization details
	//instance.Info().Bool("debug_mode", debug).Msg("Logger initialized")
	return instance
}

// GetLogger returns the logger instance
func GetLogger() zerolog.Logger {
	return instance
}

// Helper functions for consistent logging
func Info() *zerolog.Event {
	return instance.Info()
}

func Error() *zerolog.Event {
	return instance.Error()
}

func Debug() *zerolog.Event {
	return instance.Debug()
}

func Warn() *zerolog.Event {
	return instance.Warn()
}

func Fatal() *zerolog.Event {
	return instance.Fatal()
}
