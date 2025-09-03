package logger

import (
	"os"
	"path/filepath"
	"strings"
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
		// Generate timestamped log file name
		t := time.Now()
		logFileName := t.Format("sync-20060102150405.log")

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
		file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Fallback to console only if file cannot be opened
			instance = zerolog.New(consoleWriter).
				Level(zerolog.InfoLevel).
				With().
				Timestamp().
				Logger()
			instance.Error().Err(err).Str("file", logFileName).Msg("Failed to open log file, logging to console only")
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

		// Clean old log files (older than 15 days)
		cleanOldLogs(15)
	})

	// Log initialization details
	instance.Info().Bool("debug_mode", debug).Msg("Logger initialized")
	return instance
}

// cleanOldLogs removes log files older than the specified number of days
func cleanOldLogs(days int) {
	log := GetLogger() // Safe since instance is set

	dir := "."
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Warn().Err(err).Str("dir", dir).Msg("Failed to read directory for cleaning old logs")
		return
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	deletedCount := 0

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if strings.HasPrefix(name, "sync-") && strings.HasSuffix(name, ".log") {
			dateStr := strings.TrimPrefix(strings.TrimSuffix(name, ".log"), "sync-")
			dt, err := time.Parse("20060102150405", dateStr)
			if err != nil {
				log.Debug().Str("file", name).Err(err).Msg("Failed to parse log file date")
				continue
			}
			if dt.Before(cutoff) {
				filePath := filepath.Join(dir, name)
				if err := os.Remove(filePath); err != nil {
					log.Warn().Err(err).Str("file", filePath).Msg("Failed to delete old log file")
				} else {
					deletedCount++
					log.Debug().Str("file", filePath).Msg("Deleted old log file")
				}
			}
		}
	}

	if deletedCount > 0 {
		log.Info().Int("deleted", deletedCount).Msg("Cleaned old log files")
	}
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
