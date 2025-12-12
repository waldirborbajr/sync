package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/waldirborbajr/sync/logger"
)

// Config holds database connection parameters and pricing configuration
type Config struct {
	FirebirdUser     string
	FirebirdPassword string
	FirebirdHost     string
	FirebirdPath     string
	MySQLUser        string
	MySQLPassword    string
	MySQLHost        string
	MySQLPort        string
	MySQLDatabase    string
	Lucro            float64
	Parc3x           float64
	Parc6x           float64
	Parc10x          float64
	DebugMode        bool // Novo campo para modo debug

	// Update settings
	UpdateCheckURL     string // Endpoint returning latest version info (JSON: {"version":"v1.2.3","url":"https://..."})
	AutoUpdate         bool   // If true, will attempt to download the update automatically
	UpdateDownloadDir  string // Directory to save downloaded update
}

// LoadConfig loads environment variables from .env file
func LoadConfig() (Config, error) {
	log := logger.GetLogger()

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Error().Err(err).Msg("Error loading .env file")
		return Config{}, fmt.Errorf("error loading .env file: %w", err)
	}
	log.Info().Msg(".env file loaded successfully")

	// Parse float values with defaults
	lucro, err := strconv.ParseFloat(os.Getenv("LUCRO"), 64)
	if err != nil {
		log.Warn().Err(err).Msg("Invalid LUCRO value, using default")
	}
	parc3x, err := strconv.ParseFloat(os.Getenv("PARC3X"), 64)
	if err != nil {
		log.Warn().Err(err).Msg("Invalid PARC3X value, using default")
	}
	parc6x, err := strconv.ParseFloat(os.Getenv("PARC6X"), 64)
	if err != nil {
		log.Warn().Err(err).Msg("Invalid PARC6X value, using default")
	}
	parc10x, err := strconv.ParseFloat(os.Getenv("PARC10X"), 64)
	if err != nil {
		log.Warn().Err(err).Msg("Invalid PARC10X value, using default")
	}

	// Parse debug mode
	debugMode, err := strconv.ParseBool(os.Getenv("DEBUG_MODE"))
	if err != nil {
		log.Warn().Err(err).Str("DEBUG_MODE", os.Getenv("DEBUG_MODE")).Msg("Invalid DEBUG_MODE value, defaulting to false")
	}

	// Parse update settings
	autoUpdate, err := strconv.ParseBool(os.Getenv("AUTO_UPDATE"))
	if err != nil {
		log.Warn().Err(err).Str("AUTO_UPDATE", os.Getenv("AUTO_UPDATE")).Msg("Invalid AUTO_UPDATE value, defaulting to false")
	}

	// Set defaults if not provided
	if lucro == 0 {
		lucro = 40.00
	}
	if parc3x == 0 {
		parc3x = 5.00
	}
	if parc6x == 0 {
		parc6x = 10.00
	}
	if parc10x == 0 {
		parc10x = 15.00
	}

	updateDir := os.Getenv("UPDATE_DOWNLOAD_DIR")
	if updateDir == "" {
		updateDir = "."
	}

	cfg := Config{
		FirebirdUser:     os.Getenv("FIREBIRD_USER"),
		FirebirdPassword: os.Getenv("FIREBIRD_PASSWORD"),
		FirebirdHost:     os.Getenv("FIREBIRD_HOST"),
		FirebirdPath:     os.Getenv("FIREBIRD_PATH"),
		MySQLUser:        os.Getenv("MYSQL_USER"),
		MySQLPassword:    os.Getenv("MYSQL_PASSWORD"),
		MySQLHost:        os.Getenv("MYSQL_HOST"),
		MySQLPort:        os.Getenv("MYSQL_PORT"),
		MySQLDatabase:    os.Getenv("MYSQL_DATABASE"),
		Lucro:            lucro,
		Parc3x:           parc3x,
		Parc6x:           parc6x,
		Parc10x:          parc10x,
		DebugMode:        debugMode,
		UpdateCheckURL:   os.Getenv("UPDATE_CHECK_URL"),
		AutoUpdate:       autoUpdate,
		UpdateDownloadDir: updateDir,
	}

	// Validate required fields
	if cfg.FirebirdUser == "" || cfg.FirebirdPassword == "" || cfg.FirebirdHost == "" || cfg.FirebirdPath == "" {
		log.Error().Msg("Missing required Firebird environment variables")
		return Config{}, fmt.Errorf("missing required Firebird environment variables")
	}
	if cfg.MySQLUser == "" || cfg.MySQLPassword == "" || cfg.MySQLHost == "" || cfg.MySQLPort == "" || cfg.MySQLDatabase == "" {
		log.Error().Msg("Missing required MySQL environment variables")
		return Config{}, fmt.Errorf("missing required MySQL environment variables")
	}

	// Log loaded configuration for troubleshooting
	log.Debug().
		Str("FIREBIRD_USER", cfg.FirebirdUser).
		Str("FIREBIRD_HOST", cfg.FirebirdHost).
		Str("FIREBIRD_PATH", cfg.FirebirdPath).
		Str("MYSQL_USER", cfg.MySQLUser).
		Str("MYSQL_HOST", cfg.MySQLHost).
		Str("MYSQL_PORT", cfg.MySQLPort).
		Str("MYSQL_DATABASE", cfg.MySQLDatabase).
		Float64("LUCRO", cfg.Lucro).
		Float64("PARC3X", cfg.Parc3x).
		Float64("PARC6X", cfg.Parc6x).
		Float64("PARC10X", cfg.Parc10x).
		Bool("DEBUG_MODE", cfg.DebugMode).
		Str("UPDATE_CHECK_URL", cfg.UpdateCheckURL).
		Bool("AUTO_UPDATE", cfg.AutoUpdate).
		Str("UPDATE_DOWNLOAD_DIR", cfg.UpdateDownloadDir).
		Msg("Configuration loaded")

	return cfg, nil
}

// GetFirebirdDSN constructs the Firebird connection string
func (c Config) GetFirebirdDSN() string {
	return fmt.Sprintf("%s:%s@%s/%s", c.FirebirdUser, c.FirebirdPassword, c.FirebirdHost, c.FirebirdPath)
}

// GetMySQLDSN constructs the MySQL connection string
func (c Config) GetMySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		c.MySQLUser, c.MySQLPassword, c.MySQLHost, c.MySQLPort, c.MySQLDatabase)
}
