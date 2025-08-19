package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds database connection parameters
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
}

// LoadConfig loads environment variables from .env file
func LoadConfig() (Config, error) {
	if err := godotenv.Load(); err != nil {
		return Config{}, fmt.Errorf("error loading .env file: %w", err)
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
