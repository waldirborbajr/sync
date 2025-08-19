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
		return nil, fmt.Errorf("MySQL database is offline or inaccessible: %w", err)
	}
	return db, nil
}

// GetSemaphoreSize retrieves MySQL max_connections and calculates semaphore size
func GetSemaphoreSize(db *sql.DB) (semaphoreSize, maxConnections int, err error) {
	semaphoreSize = 20 // Default fallback
	var variableName string
	err = db.QueryRow("SHOW VARIABLES LIKE 'max_connections'").Scan(&variableName, &maxConnections)
	if err != nil {
		return semaphoreSize, 0, fmt.Errorf("error retrieving max_connections: %w", err)
	}
	// Use 50% of max_connections, with a minimum of 10 and maximum of 50
	semaphoreSize = maxConnections / 2
	if semaphoreSize < 10 {
		semaphoreSize = 10
	} else if semaphoreSize > 50 {
		semaphoreSize = 50
	}
	return semaphoreSize, maxConnections, nil
}

// PrepareStatements prepares MySQL update and insert statements
func PrepareStatements(db *sql.DB) (*sql.Stmt, *sql.Stmt, error) {
	updateStmt, err := db.Prepare(`
        UPDATE estoque_produtos
        SET descricao = ?, quantidade = ?
        WHERE id_clipp = ?
    `)
	if err != nil {
		return nil, nil, fmt.Errorf("error preparing MySQL update statement: %w", err)
	}
	insertStmt, err := db.Prepare(`
        INSERT INTO estoque_produtos (id_clipp, descricao, quantidade)
        VALUES (?, ?, ?)
    `)
	if err != nil {
		if closeErr := updateStmt.Close(); closeErr != nil {
			log.Printf("error closing MySQL update statement: %v", closeErr)
		}
		return nil, nil, fmt.Errorf("error preparing MySQL insert statement: %w", err)
	}
	return updateStmt, insertStmt, nil
}
