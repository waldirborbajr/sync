package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"
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

	// Get dynamic semaphore size and max_allowed_packet
	semaphoreSize, maxConnections, maxAllowedPacket, err := db.GetSemaphoreSize(mysqlConn)
	if err != nil {
		log.Printf("Error retrieving MySQL variables, using defaults: %v", err)
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

	// Variável para armazenar o batch size calculado
	var batchSize int

	// Process Firebird rows and synchronize with MySQL
	err = processor.ProcessRows(firebirdConn, mysqlConn, updateStmt, insertStmt, semaphoreSize, maxAllowedPacket, &insertedCount, &updatedCount, &ignoredCount, &batchSize)
	if err != nil {
		log.Fatal("Error processing rows:", err)
	}

	// Calculate elapsed time and throughput
	elapsedTime := time.Since(startTime)
	totalRows := insertedCount + updatedCount + ignoredCount

	// Calculate throughput
	rowsPerSecond := 0.0
	mbPerSecond := 0.0
	if elapsedTime.Seconds() > 0 {
		rowsPerSecond = float64(totalRows) / elapsedTime.Seconds()
		megabytesProcessed := float64(totalRows*200) / (1024 * 1024) // Estimativa de 200 bytes por linha
		mbPerSecond = megabytesProcessed / elapsedTime.Seconds()
	}

	// Print summary melhorado
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("SYNCHRONIZATION SUMMARY")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("MySQL max_connections: \033[1;32m%d\033[0m\n", maxConnections)
	fmt.Printf("MySQL max_allowed_packet: \033[1;32m%d MB\033[0m\n", maxAllowedPacket/(1024*1024))
	fmt.Printf("Used semaphore size: \033[1;32m%d/%d\033[0m\n", semaphoreSize, maxConnections)
	fmt.Printf("Batch size: \033[1;32m%d rows\033[0m\n", batchSize)
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Total rows processed: \033[1;32m%d\033[0m\n", totalRows)
	fmt.Printf("Total rows inserted: \033[1;32m%d\033[0m\n", insertedCount)
	fmt.Printf("Total rows updated: \033[1;32m%d\033[0m\n", updatedCount)
	fmt.Printf("Total rows ignored: \033[1;32m%d\033[0m\n", ignoredCount)
	fmt.Printf("Elapsed time: \033[1;32m%s\033[0m\n", elapsedTime.Round(time.Millisecond))
	fmt.Printf("Throughput: \033[1;32m%.2f rows/second\033[0m\n", rowsPerSecond)
	fmt.Printf("Data rate: \033[1;32m%.2f MB/second\033[0m\n", mbPerSecond)

	if totalRows > 0 {
		fmt.Printf("Efficiency: \033[1;32m%.2f ms/row\033[0m\n", (elapsedTime.Seconds()*1000)/float64(totalRows))
	}

	// Adicionar estatísticas de memória
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Memory usage: \033[1;32m%.2f MB\033[0m\n", float64(m.Alloc)/1024/1024)
	fmt.Println(strings.Repeat("=", 70))
}
