package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/nakagami/firebirdsql"
	"github.com/waldirborbajr/sync/config"
)

// ConnectFirebird establishes a connection to the Firebird database
func ConnectFirebird(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("firebirdsql", cfg.GetFirebirdDSN())
	if err != nil {
		return nil, fmt.Errorf("error opening Firebird connection: %w", err)
	}
	if err = db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("error closing Firebird database connection: %v", closeErr)
		}
		return nil, fmt.Errorf("firebird database is offline or inaccessible: %w", err)
	}
	return db, nil
}

// ConnectMySQL establishes a connection to the MySQL database
func ConnectMySQL(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.GetMySQLDSN())
	if err != nil {
		return nil, fmt.Errorf("error opening MySQL connection: %w", err)
	}
	if err = db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("error closing MySQL database connection: %v", closeErr)
		}
		return nil, fmt.Errorf("mysql database is offline or inaccessible: %w", err)
	}
	return db, nil
}

// GetSemaphoreSize retrieves MySQL max_connections and max_allowed_packet
func GetSemaphoreSize(db *sql.DB) (semaphoreSize, maxConnections int, maxAllowedPacket int, err error) {
	semaphoreSize = 20                 // Default fallback
	maxAllowedPacket = 4 * 1024 * 1024 // Default 4MB

	// Get max_connections
	var variableName string
	err = db.QueryRow("SHOW VARIABLES LIKE 'max_connections'").Scan(&variableName, &maxConnections)
	if err != nil {
		return semaphoreSize, 0, maxAllowedPacket, fmt.Errorf("error retrieving max_connections: %w", err)
	}

	// Get max_allowed_packet
	var packetSize int
	err = db.QueryRow("SHOW VARIABLES LIKE 'max_allowed_packet'").Scan(&variableName, &packetSize)
	if err == nil {
		maxAllowedPacket = packetSize
	} else {
		log.Printf("Warning: error retrieving max_allowed_packet: %v", err)
	}

	// Use 75% de max_connections, com mínimo de 10 e máximo de 100
	semaphoreSize = int(float64(maxConnections) * 0.75)
	if semaphoreSize < 10 {
		semaphoreSize = 10
	} else if semaphoreSize > 100 {
		semaphoreSize = 100
	}

	return semaphoreSize, maxConnections, maxAllowedPacket, nil
}

// PrepareStatements prepares MySQL update and insert statements with new fields
func PrepareStatements(db *sql.DB) (*sql.Stmt, *sql.Stmt, error) {
	updateStmt, err := db.Prepare(`
        UPDATE TB_ESTOQUE
        SET DESCRICAO = ?, QTD_ATUAL = ?, PRC_CUSTO = ?, PRC_DOLAR = ?, 
            PRC_VENDA = ?, PRC_3X = ?, PRC_6X = ?, PRC_10X = ?
        WHERE ID_ESTOQUE = ?
    `)
	if err != nil {
		return nil, nil, fmt.Errorf("error preparing MySQL update statement: %w", err)
	}
	insertStmt, err := db.Prepare(`
        INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, QTD_ATUAL, PRC_CUSTO, PRC_DOLAR, 
                               PRC_VENDA, PRC_3X, PRC_6X, PRC_10X)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		if closeErr := updateStmt.Close(); closeErr != nil {
			log.Printf("error closing MySQL update statement: %v", closeErr)
		}
		return nil, nil, fmt.Errorf("error preparing MySQL insert statement: %w", err)
	}
	return updateStmt, insertStmt, nil
}
