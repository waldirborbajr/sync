package db

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/waldirborbajr/sync/config"
	"github.com/waldirborbajr/sync/logger"
	_ "modernc.org/sqlite"
)

// ConnectFirebirdDev creates an SQLite database to mock Firebird for development
func ConnectFirebirdDev(cfg config.Config) (*sql.DB, error) {
	log := logger.GetLogger()

	dbPath := "./dev_firebird.db"

	// Check if database exists
	dbExists := false
	if _, err := os.Stat(dbPath); err == nil {
		dbExists = true
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening SQLite Firebird mock: %w", err)
	}

	if err = db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("Error closing SQLite Firebird mock")
		}
		return nil, fmt.Errorf("SQLite Firebird mock is not accessible: %w", err)
	}

	// Initialize schema if database is new
	if !dbExists {
		log.Info().Msg("Initializing Firebird mock database schema")
		if err := initFirebirdSchema(db); err != nil {
			db.Close()
			return nil, fmt.Errorf("error initializing Firebird schema: %w", err)
		}
	}

	log.Info().
		Str("path", dbPath).
		Msg("SQLite Firebird mock connected successfully (DEV MODE)")
	return db, nil
}

// ConnectMySQLDev creates an SQLite database to mock MySQL for development
func ConnectMySQLDev(cfg config.Config) (*sql.DB, error) {
	log := logger.GetLogger()

	dbPath := "./dev_mysql.db"

	// Check if database exists
	dbExists := false
	if _, err := os.Stat(dbPath); err == nil {
		dbExists = true
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening SQLite MySQL mock: %w", err)
	}

	if err = db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("Error closing SQLite MySQL mock")
		}
		return nil, fmt.Errorf("SQLite MySQL mock is not accessible: %w", err)
	}

	// Initialize schema if database is new
	if !dbExists {
		log.Info().Msg("Initializing MySQL mock database schema")
		if err := initMySQLSchema(db); err != nil {
			db.Close()
			return nil, fmt.Errorf("error initializing MySQL schema: %w", err)
		}
		log.Info().Msg("MySQL mock database initialized with empty table")
	}

	log.Info().
		Str("path", dbPath).
		Msg("SQLite MySQL mock connected successfully (DEV MODE)")
	return db, nil
}

// initFirebirdSchema creates tables and sample data for Firebird mock
func initFirebirdSchema(db *sql.DB) error {
	log := logger.GetLogger()

	// Try to load from dev_firebird_data.sql file
	sqlFilePath := "./dev_firebird_data.sql"
	if sqlContent, err := os.ReadFile(sqlFilePath); err == nil {
		log.Info().Str("file", sqlFilePath).Msg("Loading Firebird schema from SQL file")

		// Execute the SQL file content
		if _, err := db.Exec(string(sqlContent)); err != nil {
			return fmt.Errorf("error executing SQL from file %s: %w", sqlFilePath, err)
		}

		log.Info().Msg("Firebird schema loaded successfully from SQL file (110 products)")
		return nil
	}

	// Fallback: Use minimal hardcoded sample data if SQL file doesn't exist
	log.Warn().Str("file", sqlFilePath).Msg("SQL file not found, using minimal sample data")

	schema := `
	CREATE TABLE IF NOT EXISTS TB_ESTOQUE (
		ID_ESTOQUE INTEGER PRIMARY KEY,
		DESCRICAO TEXT NOT NULL,
		PRC_CUSTO REAL,
		STATUS TEXT DEFAULT 'A'
	);

	CREATE TABLE IF NOT EXISTS TB_EST_PRODUTO (
		ID_IDENTIFICADOR INTEGER PRIMARY KEY,
		QTD_ATUAL REAL DEFAULT 0,
		FOREIGN KEY (ID_IDENTIFICADOR) REFERENCES TB_ESTOQUE(ID_ESTOQUE)
	);

	CREATE TABLE IF NOT EXISTS TB_EST_INDEXADOR (
		ID_ESTOQUE INTEGER PRIMARY KEY,
		VALOR REAL DEFAULT 0,
		FOREIGN KEY (ID_ESTOQUE) REFERENCES TB_ESTOQUE(ID_ESTOQUE)
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("error creating Firebird mock schema: %w", err)
	}

	// Insert minimal sample data
	sampleData := `
	-- Sample products
	INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, PRC_CUSTO, STATUS) VALUES
		(1, 'Product A - Sample Item', 100.00, 'A'),
		(2, 'Product B - Test Widget', 250.50, 'A'),
		(3, 'Product C - Development Kit', 500.00, 'A'),
		(4, 'Product D - Mock Component', 75.25, 'A'),
		(5, 'Product E - Testing Tool', 150.00, 'A'),
		(17973, 'Special Test Product', 1000.00, 'A'),
		(100, 'Inactive Product', 200.00, 'I');

	-- Quantities
	INSERT INTO TB_EST_PRODUTO (ID_IDENTIFICADOR, QTD_ATUAL) VALUES
		(1, 50),
		(2, 25),
		(3, 10),
		(4, 100),
		(5, 35),
		(17973, 5),
		(100, 0);

	-- USD values
	INSERT INTO TB_EST_INDEXADOR (ID_ESTOQUE, VALOR) VALUES
		(1, 18.50),
		(2, 46.20),
		(3, 92.40),
		(4, 13.90),
		(5, 27.70),
		(17973, 184.80),
		(100, 36.90);
	`

	if _, err := db.Exec(sampleData); err != nil {
		return fmt.Errorf("error inserting Firebird sample data: %w", err)
	}

	log.Info().Msg("Firebird schema initialized with minimal sample data (7 products)")
	return nil
}

// initMySQLSchema creates the TB_ESTOQUE table for MySQL mock
func initMySQLSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS TB_ESTOQUE (
		ID_ESTOQUE INTEGER PRIMARY KEY,
		DESCRICAO TEXT NOT NULL,
		QTD_ATUAL REAL DEFAULT 0,
		PRC_CUSTO REAL DEFAULT 0,
		PRC_DOLAR REAL DEFAULT 0,
		PRC_VENDA REAL DEFAULT 0,
		PRC_3X REAL DEFAULT 0,
		PRC_6X REAL DEFAULT 0,
		PRC_10X REAL DEFAULT 0
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("error creating MySQL mock schema: %w", err)
	}

	// Start with empty table - data will be synced from Firebird mock
	return nil
}
