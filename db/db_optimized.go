package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/nakagami/firebirdsql"
	"github.com/waldirborbajr/sync/config"
	"github.com/waldirborbajr/sync/logger"
)

// ConnectFirebirdOptimized establishes an optimized connection to the Firebird database
func ConnectFirebirdOptimized(cfg config.Config) (*sql.DB, error) {
	log := logger.GetLogger()

	db, err := sql.Open("firebirdsql", cfg.GetFirebirdDSN())
	if err != nil {
		return nil, fmt.Errorf("error opening Firebird connection: %w", err)
	}

	// Optimize connection pool for Firebird
	db.SetMaxOpenConns(25)                  // Limit concurrent connections to Firebird
	db.SetMaxIdleConns(10)                  // Keep some idle connections ready
	db.SetConnMaxLifetime(30 * time.Minute) // Rotate connections

	if err = db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("Error closing Firebird database connection")
		}
		return nil, fmt.Errorf("firebird database is offline or inaccessible: %w", err)
	}

	log.Debug().
		Int("max_open_conns", 25).
		Int("max_idle_conns", 10).
		Msg("Firebird database connected successfully with optimized settings")
	return db, nil
}

// ConnectMySQLOptimized establishes an optimized connection to the MySQL database
func ConnectMySQLOptimized(cfg config.Config) (*sql.DB, error) {
	log := logger.GetLogger()

	// Add connection parameters for better performance
	dsn := cfg.GetMySQLDSN() + "&writeTimeout=10s&readTimeout=30s&timeout=5s"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening MySQL connection: %w", err)
	}

	// Get max_connections first to set appropriate pool size
	var variableName string
	var maxConnections int
	err = db.QueryRow("SHOW VARIABLES LIKE 'max_connections'").Scan(&variableName, &maxConnections)
	if err != nil {
		log.Warn().Err(err).Msg("Could not read max_connections, using default pool size")
		maxConnections = 200
	}

	// Optimize connection pool for MySQL
	// Use 80% of max_connections for connection pool
	maxOpenConns := int(float64(maxConnections) * 0.8)
	if maxOpenConns > 200 {
		maxOpenConns = 200 // Cap at reasonable limit
	}
	if maxOpenConns < 50 {
		maxOpenConns = 50 // Minimum for good performance
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns / 2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	if err = db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("Error closing MySQL database connection")
		}
		return nil, fmt.Errorf("mysql database is offline or inaccessible: %w", err)
	}

	log.Debug().
		Int("max_connections", maxConnections).
		Int("max_open_conns", maxOpenConns).
		Int("max_idle_conns", maxOpenConns/2).
		Msg("MySQL database connected successfully with optimized settings")
	return db, nil
}

// PrepareBulkStatements prepares optimized bulk statements for MySQL
func PrepareBulkStatements(db *sql.DB, batchSize int) (*sql.Stmt, error) {
	log := logger.GetLogger()

	// Single update statement (will be executed per row, but optimized)
	updateStmt, err := db.Prepare(`
        UPDATE TB_ESTOQUE
        SET DESCRICAO = ?, QTD_ATUAL = ?, PRC_CUSTO = ?, PRC_DOLAR = ?, 
            PRC_VENDA = ?, PRC_3X = ?, PRC_6X = ?, PRC_10X = ?
        WHERE ID_ESTOQUE = ?
    `)
	if err != nil {
		return nil, fmt.Errorf("error preparing MySQL update statement: %w", err)
	}

	log.Debug().
		Int("batch_size", batchSize).
		Msg("MySQL bulk statements prepared successfully")

	return updateStmt, nil
}
