package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
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
}

// LoadConfig loads environment variables from .env file
func LoadConfig() (Config, error) {
	if err := godotenv.Load(); err != nil {
		return Config{}, fmt.Errorf("error loading .env file: %w", err)
	}

	// Parse float values with defaults
	lucro, _ := strconv.ParseFloat(os.Getenv("LUCRO"), 64)
	parc3x, _ := strconv.ParseFloat(os.Getenv("PARC3X"), 64)
	parc6x, _ := strconv.ParseFloat(os.Getenv("PARC6X"), 64)
	parc10x, _ := strconv.ParseFloat(os.Getenv("PARC10X"), 64)

	// Parse debug mode
	debugMode, _ := strconv.ParseBool(os.Getenv("DEBUG_MODE"))

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
	}

	// Validate required fields
	if cfg.FirebirdUser == "" || cfg.FirebirdPassword == "" || cfg.FirebirdHost == "" || cfg.FirebirdPath == "" {
		return Config{}, fmt.Errorf("missing required Firebird environment variables")
	}
	if cfg.MySQLUser == "" || cfg.MySQLPassword == "" || cfg.MySQLHost == "" || cfg.MySQLPort == "" || cfg.MySQLDatabase == "" {
		return Config{}, fmt.Errorf("missing required MySQL environment variables")
	}

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
