package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/waldirborbajr/sync/config"
	"github.com/waldirborbajr/sync/db"
	"github.com/waldirborbajr/sync/logger"
	"github.com/waldirborbajr/sync/processor"
)

// version is set at build time using -ldflags="-X main.version=VERSION"
var version string

func main() {
	// Load configuration from .env
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Error loading configuration")
	}

	// Initialize logger
	log := logger.InitLogger(cfg.DebugMode)

	fmt.Printf("\nSynC Firebird x MySQL v%s\n\n", version)

	// Track counts and start time
	var insertedCount, updatedCount, ignoredCount int
	startTime := time.Now()

	// Connect to Firebird database
	firebirdConn, err := db.ConnectFirebird(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Firebird")
	}
	defer func() {
		if err := firebirdConn.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing Firebird database connection")
		}
	}()

	// Connect to MySQL database
	mysqlConn, err := db.ConnectMySQL(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MySQL")
	}
	defer func() {
		if err := mysqlConn.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing MySQL database connection")
		}
	}()

	// Otimizações do MySQL
	_, err = mysqlConn.Exec("SET unique_checks=0")
	if err != nil {
		log.Warn().Err(err).Msg("Could not set unique_checks=0")
	}
	_, err = mysqlConn.Exec("SET foreign_key_checks=0")
	if err != nil {
		log.Warn().Err(err).Msg("Could not set foreign_key_checks=0")
	}

	// Get dynamic semaphore size and max_allowed_packet
	semaphoreSize, maxConnections, maxAllowedPacket, err := db.GetSemaphoreSize(mysqlConn)
	if err != nil {
		log.Warn().Err(err).Msg("Error retrieving MySQL variables, using defaults")
	}

	// Prepare MySQL statements
	updateStmt, insertStmt, err := db.PrepareStatements(mysqlConn)
	if err != nil {
		log.Fatal().Err(err).Msg("Error preparing MySQL statements")
	}
	defer func() {
		if err := updateStmt.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing MySQL update statement")
		}
	}()
	defer func() {
		if err := insertStmt.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing MySQL insert statement")
		}
	}()

	// Variáveis para estatísticas
	var batchSize int
	stats := &processor.ProcessingStats{}

	// Process Firebird rows
	err = processor.ProcessRows(firebirdConn, mysqlConn, updateStmt, insertStmt, semaphoreSize, maxAllowedPacket, &insertedCount, &updatedCount, &ignoredCount, &batchSize, stats, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Error processing rows")
	}

	// Restaurar configurações do MySQL
	_, err = mysqlConn.Exec("SET unique_checks=1")
	if err != nil {
		log.Warn().Err(err).Msg("Could not set unique_checks=1")
	}
	_, err = mysqlConn.Exec("SET foreign_key_checks=1")
	if err != nil {
		log.Warn().Err(err).Msg("Could not set foreign_key_checks=1")
	}

	// Calculate elapsed time and throughput
	elapsedTime := time.Since(startTime)
	totalRows := insertedCount + updatedCount + ignoredCount

	// Calculate throughput
	rowsPerSecond := 0.0
	mbPerSecond := 0.0
	if elapsedTime.Seconds() > 0 {
		rowsPerSecond = float64(totalRows) / elapsedTime.Seconds()
		megabytesProcessed := float64(totalRows*200) / (1024 * 1024)
		mbPerSecond = megabytesProcessed / elapsedTime.Seconds()
	}

	// Print detailed summary
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("SYNCHRONIZATION PERFORMANCE REPORT")
	fmt.Println(strings.Repeat("=", 80))

	// Database Configuration
	fmt.Println("DATABASE CONFIGURATION:")
	fmt.Printf("  MySQL max_connections: \033[1;32m%d\033[0m\n", maxConnections)
	fmt.Printf("  MySQL max_allowed_packet: \033[1;32m%d MB\033[0m\n", maxAllowedPacket/(1024*1024))
	fmt.Printf("  Used semaphore size: \033[1;32m%d/%d\033[0m\n", semaphoreSize, maxConnections)
	fmt.Printf("  Batch size: \033[1;32m%d rows\033[0m\n", batchSize)

	// Performance Metrics
	fmt.Println("\nPERFORMANCE METRICS:")
	fmt.Printf("  Data loading time: \033[1;36m%s\033[0m\n", stats.LoadTime.Round(time.Millisecond))
	fmt.Printf("  Query execution time: \033[1;36m%s\033[0m\n", stats.QueryTime.Round(time.Millisecond))
	fmt.Printf("  Processing time: \033[1;36m%s\033[0m\n", stats.ProcessingTime.Round(time.Millisecond))
	fmt.Printf("  Procedure time: \033[1;36m%s\033[0m\n", stats.ProcedureTime.Round(time.Millisecond))
	fmt.Printf("  Total elapsed time: \033[1;36m%s\033[0m\n", elapsedTime.Round(time.Millisecond))

	// Throughput
	fmt.Printf("  Throughput: \033[1;35m%.2f rows/second\033[0m\n", rowsPerSecond)
	fmt.Printf("  Data rate: \033[1;35m%.2f MB/second\033[0m\n", mbPerSecond)
	if totalRows > 0 {
		fmt.Printf("  Efficiency: \033[1;35m%.3f ms/row\033[0m\n", (elapsedTime.Seconds()*1000)/float64(totalRows))
	}

	// Results
	fmt.Println("\nRESULTS:")
	fmt.Printf("  Total rows processed: \033[1;32m%d\033[0m\n", totalRows)
	fmt.Printf("  Rows inserted: \033[1;32m%d\033[0m\n", insertedCount)
	fmt.Printf("  Rows updated: \033[1;33m%d\033[0m\n", updatedCount)
	fmt.Printf("  Rows ignored: \033[1;34m%d\033[0m\n", ignoredCount)

	// Memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("  Memory usage: \033[1;36m%.2f MB\033[0m\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("  System memory: \033[1;36m%.2f MB\033[0m\n", float64(m.Sys)/1024/1024)

	// GC statistics
	fmt.Printf("  GC cycles: \033[1;36m%d\033[0m\n", m.NumGC)
	if m.NumGC > 0 {
		fmt.Printf("  GC pause: \033[1;36m%.2fms\033[0m\n", float64(m.PauseTotalNs)/float64(m.NumGC)/1000000)
	}

	fmt.Println(strings.Repeat("=", 80))

	// Performance recommendations
	fmt.Println("\nPERFORMANCE RECOMMENDATIONS:")
	if stats.LoadTime > 2*time.Second {
		fmt.Println("  ⚡ Consider adding indexes to MySQL TB_ESTOQUE table")
	}
	if stats.ProcessingTime > 5*time.Second {
		fmt.Println("  ⚡ Consider increasing MySQL max_connections")
	}
	if float64(updatedCount)/float64(totalRows) > 0.7 {
		fmt.Println("  ⚡ High update rate - consider optimizing comparison logic")
	}
	if m.NumGC > 10 {
		fmt.Println("  ⚡ High GC pressure - consider reducing memory allocation")
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Synchronization completed successfully in %s!", elapsedTime.Round(time.Millisecond))
}
