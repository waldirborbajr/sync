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

	// Print pricing configuration
	log.Info().Msg("Pricing Configuration:")
	log.Info().Msgf("  LUCRO: %.3f%%", cfg.Lucro)
	log.Info().Msgf("  PARC3X: %.2f%%", cfg.Parc3x)
	log.Info().Msgf("  PARC6X: %.2f%%", cfg.Parc6x)
	log.Info().Msgf("  PARC10X: %.2f%%", cfg.Parc10x)

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
	log.Info().Msg(strings.Repeat("=", 80))
	log.Info().Msg("SYNCHRONIZATION PERFORMANCE REPORT")
	log.Info().Msg(strings.Repeat("=", 80))

	// Database Configuration
	log.Info().Msg("DATABASE CONFIGURATION:")
	log.Info().Msgf("  MySQL max_connections: %d", maxConnections)
	log.Info().Msgf("  MySQL max_allowed_packet: %d MB", maxAllowedPacket/(1024*1024))
	log.Info().Msgf("  Used semaphore size: %d/%d", semaphoreSize, maxConnections)
	log.Info().Msgf("  Batch size: %d rows", batchSize)

	// Performance Metrics
	log.Info().Msg("\nPERFORMANCE METRICS:")
	log.Info().Msgf("  Data loading time: %s", stats.LoadTime.Round(time.Millisecond))
	log.Info().Msgf("  Query execution time: %s", stats.QueryTime.Round(time.Millisecond))
	log.Info().Msgf("  Processing time: %s", stats.ProcessingTime.Round(time.Millisecond))
	log.Info().Msgf("  Procedure time: %s", stats.ProcedureTime.Round(time.Millisecond))
	log.Info().Msgf("  Total elapsed time: %s", elapsedTime.Round(time.Millisecond))

	// Throughput
	log.Info().Msgf("  Throughput: %.2f rows/second", rowsPerSecond)
	log.Info().Msgf("  Data rate: %.2f MB/second", mbPerSecond)
	if totalRows > 0 {
		log.Info().Msgf("  Efficiency: %.3f ms/row", (elapsedTime.Seconds()*1000)/float64(totalRows))
	}

	// Results
	log.Info().Msg("\nRESULTS:")
	log.Info().Msgf("  Total rows processed: %d", totalRows)
	log.Info().Msgf("  Rows inserted: %d", insertedCount)
	log.Info().Msgf("  Rows updated: %d", updatedCount)
	log.Info().Msgf("  Rows ignored: %d", ignoredCount)

	// Memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Info().Msgf("  Memory usage: %.2f MB", float64(m.Alloc)/(1024*1024))

	log.Info().Msg(strings.Repeat("=", 80))
	log.Info().Msgf("Synchronization completed successfully in %s!", elapsedTime.Round(time.Millisecond))
}
