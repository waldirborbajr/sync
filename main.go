package main

import (
	"fmt"
	"log"
	"time"

	"github.com/waldirborbajr/sync/config"
	"github.com/waldirborbajr/sync/db"
	"github.com/waldirborbajr/sync/processor"
)

// version is set at build time using -ldflags="-X main.version=VERSION"
var version string

func main() {
	fmt.Printf("\nSynC Firebird x MySQL v%s\n\n", version)

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
		log.Fatal(err) // Includes "Firebird database is offline or inaccessible" if applicable
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
		log.Fatal(err) // Includes "MySQL database is offline or inaccessible" if applicable
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
	fmt.Printf("Used semaphore size: \033[1;32m%d/%d\033[0m\n", semaphoreSize, maxConnections)
	fmt.Printf("Total rows inserted: \033[1;32m%d\033[0m\n", insertedCount)
	fmt.Printf("Total rows updated: \033[1;32m%d\033[0m\n", updatedCount)
	fmt.Printf("Total rows ignored: \033[1;32m%d\033[0m\n", ignoredCount)
	fmt.Printf("Elapsed time: \033[1;32m%s\033[0m\n", elapsedTime)
}
