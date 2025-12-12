package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"github.com/rs/zerolog"
)

var (
	once     sync.Once
	instance zerolog.Logger
)

// InitLogger initializes the logger with configurations for console and file output
func InitLogger(debug bool) zerolog.Logger {
	once.Do(func() {
		// Ensure logs directory exists
		logsDir := "logs"
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create logs dir: %v\n", err)
			logsDir = "."
		}

		// Generate timestamped log file name
		t := time.Now()
		logFileName := t.Format("sync-20060102150405.log")
		logFilePath := filepath.Join(logsDir, logFileName)

		// Configure console output
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			FormatLevel: func(i interface{}) string {
				s := strings.ToUpper(fmt.Sprint(i))
				s = strings.TrimSpace(s)
				if s == "" {
					return s
				}
				switch s {
				case "DEBUG":
					return "\033[36m[DBG]\033[0m" // cyan
				case "INFO":
					return "\033[32m[INF]\033[0m" // green
				case "WARN":
					return "\033[33m[WRN]\033[0m" // yellow
				case "ERROR":
					return "\033[31m[ERR]\033[0m" // red
				case "FATAL":
					return "\033[31m[FAT]\033[0m" // red
				default:
					return s
				}
			},
			FormatMessage: func(i interface{}) string {
				return fmt.Sprint(i)
			},
		}

		// Configure rotating file output using lumberjack
	// Allow overrides via environment variables
	maxSizeMB := 50
	if s := os.Getenv("LOG_MAX_SIZE_MB"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			maxSizeMB = v
		}
	}
	maxBackups := 7
	if s := os.Getenv("LOG_MAX_BACKUPS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			maxBackups = v
		}
	}
	maxAge := 15
	if s := os.Getenv("LOG_MAX_AGE_DAYS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			maxAge = v
		}
	}
	compress := true
	if s := os.Getenv("LOG_COMPRESS"); s != "" {
		if v, err := strconv.ParseBool(s); err == nil {
			compress = v
		}
	}

	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   compress,
	}

	// MultiWriter: runtime logs go to both console and rotating file
	multiWriter := zerolog.MultiLevelWriter(consoleWriter, lumberjackLogger)

		// Configure logger level and fields based on debug mode
		level := zerolog.InfoLevel
		if debug {
			level = zerolog.DebugLevel
		}

		zerolog.SetGlobalLevel(level)
		zerolog.TimeFieldFormat = time.RFC3339

		baseLogger := zerolog.New(multiWriter).
			Level(level).
			With().
			Str("app", "sync").
			Str("goos", runtime.GOOS).
			Str("goarch", runtime.GOARCH).
			Timestamp().
			Logger()

		if debug {
			// add caller and stack for debug level
			baseLogger = baseLogger.With().Caller().Logger()
		}

		instance = baseLogger

		// Clean old log files (older than 15 days)
		cleanOldLogs(logsDir, 15)

		// Log initialization details only to file (not console)
		fileLogger := zerolog.New(file).
			Level(level).
			With().
			Str("app", "sync").
			Timestamp().
			Logger()
		fileLogger.Info().Bool("debug_mode", debug).Msg("Logger initialized")
	})

	return instance
}

// cleanOldLogs removes log files older than the specified number of days in a directory
func cleanOldLogs(dir string, days int) {
	log := GetLogger() // Safe since instance is set

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
