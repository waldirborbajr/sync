package main

import (
	"fmt"
	"log"
	"time"

	"github.com/waldirborbajr/sync/config"
	"github.com/waldirborbajr/sync/db"
	"github.com/waldirborbajr/sync/processor"
)

func main() {
	// Track counts and start time
	var insertedCount, updatedCount, ignoredCount int
	startTime := time.Now()

	// Load configuration from .env
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error loading configuration:", err)
	}

	// Connect to Firebird database
	firebirdConn, err := db.ConnectFirebird(cfg)
	if err != nil {
		log.Fatal("Error connecting to Firebird:", err)
	}
	defer func() {
		if err := firebirdConn.Close(); err != nil {
			log.Printf("Error closing Firebird database connection: %v", err)
		}
	}()
	fmt.Println("Connected to Firebird database")

	// Connect to MySQL database
	mysqlConn, err := db.ConnectMySQL(cfg)
	if err != nil {
		log.Fatal("Error connecting to MySQL:", err)
	}
	defer func() {
		if err := mysqlConn.Close(); err != nil {
			log.Printf("Error closing MySQL database connection: %v", err)
		}
	}()
	fmt.Println("Connected to MySQL database")

	// Get dynamic semaphore size
	semaphoreSize, maxConnections, err := db.GetSemaphoreSize(mysqlConn)
	if err != nil {
		log.Printf("Error retrieving MySQL max_connections, using default semaphore size %d: %v", semaphoreSize, err)
	}
	fmt.Printf("Using semaphore size: %d (based on MySQL max_connections: %d)\n", semaphoreSize, maxConnections)

	// Prepare MySQL statements
	updateStmt, insertStmt, err := db.PrepareStatements(mysqlConn)
	if err != nil {
		log.Fatal("Error preparing MySQL statements:", err)
	}
	defer func() {
		if err := updateStmt.Close(); err != nil {
			log.Printf("Error closing MySQL update statement: %v", err)
		}
	}()
	defer func() {
		if err := insertStmt.Close(); err != nil {
			log.Printf("Error closing MySQL insert statement: %v", err)
		}
	}()

	// Process Firebird rows and synchronize with MySQL
	err = processor.ProcessRows(firebirdConn, mysqlConn, updateStmt, insertStmt, semaphoreSize, &insertedCount, &updatedCount, &ignoredCount)
	if err != nil {
		log.Fatal("Error processing rows:", err)
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)

	// Print summary
	fmt.Printf("Data synchronization completed.\n")
	fmt.Printf("Total de linhas inseridas: %d\n", insertedCount)
	fmt.Printf("Total de linhas alteradas: %d\n", updatedCount)
	fmt.Printf("Total de linhas ignoradas: %d\n", ignoredCount)
	fmt.Printf("Total de semáforos utilizados: %d\n", semaphoreSize)
	fmt.Printf("Tempo decorrido: %s\n", elapsedTime)
}
