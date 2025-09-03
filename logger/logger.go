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

// InitLogger inicializa o logger com configurações padrão
func InitLogger(debug bool) zerolog.Logger {
	once.Do(func() {
		// Configuração de output
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			FormatLevel: func(i interface{}) string {
				return ""
			},
			FormatMessage: func(i interface{}) string {
				return ""
			},
		}

		// Nível de log baseado no modo debug
		level := zerolog.InfoLevel
		if debug {
			level = zerolog.DebugLevel
		}

		instance = zerolog.New(output).
			Level(level).
			With().
			Timestamp().
			Logger()
	})
	return instance
}

// GetLogger retorna a instância do logger
func GetLogger() zerolog.Logger {
	return instance
}

// Helper functions para logging consistente
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