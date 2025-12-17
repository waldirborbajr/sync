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

// runProcessing orchestrates DB connections, statement preparation and processing rows.
func runProcessing(cfg config.Config) (inserted, updated, ignored, batchSize int, stats *processor.ProcessingStats, elapsed time.Duration, maxConnections int, maxAllowedPacket int, err error) {
	log := logger.GetLogger()
	// Connect to Firebird
	firebirdConn, err := db.ConnectFirebird(cfg)
	if err != nil {
		return 0, 0, 0, 0, nil, 0, 0, 0, err
	}
	defer func() {
		if firebirdConn != nil {
			if closeErr := firebirdConn.Close(); closeErr != nil {
				log.Error().Err(closeErr).Msg("Error closing Firebird database connection")
			}
		}
	}()

	// Connect to MySQL
	mysqlConn, err := db.ConnectMySQL(cfg)
	if err != nil {
		return 0, 0, 0, 0, nil, 0, 0, 0, err
	}
	defer func() {
		if mysqlConn != nil {
			if closeErr := mysqlConn.Close(); closeErr != nil {
				log.Error().Err(closeErr).Msg("Error closing MySQL database connection")
			}
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

	// Prepare statements
	updateStmt, insertStmt, err := db.PrepareStatements(mysqlConn)
	if err != nil {
		return 0, 0, 0, 0, nil, 0, 0, 0, err
	}
	defer func() {
		if updateStmt != nil {
			if cerr := updateStmt.Close(); cerr != nil {
				log.Error().Err(cerr).Msg("Error closing MySQL update statement")
			}
		}
		if insertStmt != nil {
			if cerr := insertStmt.Close(); cerr != nil {
				log.Error().Err(cerr).Msg("Error closing MySQL insert statement")
			}
		}
	}()

	// Processing
	stats = &processor.ProcessingStats{}
	startTime := time.Now()
	err = processor.ProcessRows(firebirdConn, mysqlConn, updateStmt, insertStmt, semaphoreSize, maxAllowedPacket, &inserted, &updated, &ignored, &batchSize, stats, cfg)
	if err != nil {
		return 0, 0, 0, 0, nil, 0, 0, 0, err
	}

	// Restore MySQL settings
	_, err = mysqlConn.Exec("SET unique_checks=1")
	if err != nil {
		log.Warn().Err(err).Msg("Could not set unique_checks=1")
	}
	_, err = mysqlConn.Exec("SET foreign_key_checks=1")
	if err != nil {
		log.Warn().Err(err).Msg("Could not set foreign_key_checks=1")
	}

	elapsed = time.Since(startTime)
	return inserted, updated, ignored, batchSize, stats, elapsed, maxConnections, maxAllowedPacket, nil
}

// printSummary prints the performance report
func printSummary(inserted, updated, ignored int, batchSize int, stats *processor.ProcessingStats, elapsed time.Duration, semaphoreSize, maxConnections, maxAllowedPacket int) {
	// Keep the printing logic minimal here — same formatting as before
	fmtPrintReport(inserted, updated, ignored, batchSize, stats, elapsed, semaphoreSize, maxConnections, maxAllowedPacket)
}

func fmtPrintReport(inserted, updated, ignored int, batchSize int, stats *processor.ProcessingStats, elapsed time.Duration, semaphoreSize, maxConnections, maxAllowedPacket int) {
	totalRows := inserted + updated + ignored
	rowsPerSecond := 0.0
	mbPerSecond := 0.0
	if elapsed.Seconds() > 0 {
		rowsPerSecond = float64(totalRows) / elapsed.Seconds()
		megabytesProcessed := float64(totalRows*200) / (1024 * 1024)
		mbPerSecond = megabytesProcessed / elapsed.Seconds()
	}

	fmt.Println("\n" + strings.Repeat(".", 20))
	fmt.Println("SYNCHRONIZATION PERFORMANCE REPORT")
	fmt.Println(strings.Repeat(".", 20))

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
	fmt.Printf("  Total elapsed time: \033[1;36m%s\033[0m\n", elapsed.Round(time.Millisecond))

	// Throughput
	fmt.Printf("  Throughput: \033[1;35m%.2f rows/second\033[0m\n", rowsPerSecond)
	fmt.Printf("  Data rate: \033[1;35m%.2f MB/second\033[0m\n", mbPerSecond)
	if totalRows > 0 {
		fmt.Printf("  Efficiency: \033[1;35m%.3f ms/row\033[0m\n", (elapsed.Seconds()*1000)/float64(totalRows))
	}

	// Results
	fmt.Println("\nRESULTS:")
	fmt.Printf("  Total rows processed: \033[1;32m%d\033[0m\n", totalRows)
	fmt.Printf("  Rows inserted: \033[1;32m%d\033[0m\n", inserted)
	fmt.Printf("  Rows updated: \033[1;33m%d\033[0m\n", updated)
	fmt.Printf("  Rows ignored: \033[1;34m%d\033[0m\n", ignored)

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

	fmt.Println(strings.Repeat("-", 20))

	// Performance recommendations
	fmt.Println("PERFORMANCE RECOMMENDATIONS:")
	recommendationCount := 0

	if stats.LoadTime > 2*time.Second {
		fmt.Println(redBold + "  ⚡ Consider adding indexes to MySQL TB_ESTOQUE table" + reset)
		recommendationCount++
	}
	if stats.ProcessingTime > 5*time.Second {
		fmt.Println(redBold + "  ⚡ Consider increasing MySQL max_connections" + reset)
		recommendationCount++
	}
	if float64(updated)/float64(totalRows) > 0.7 {
		fmt.Println(redBold + "  ⚡ High update rate - consider optimizing comparison logic" + reset)
		recommendationCount++
	}
	if m.NumGC > 10 {
		fmt.Println(redBold + "  ⚡ High GC pressure - consider reducing memory allocation" + reset)
		recommendationCount++
	}

	if recommendationCount == 0 {
		fmt.Println(greenBold + "  ✅ 0 issues found – running at optimal performance" + reset)
	} else {
		fmt.Printf(redBold+"  ❌ %d issues found – please review the recommendations above"+reset+"\n", recommendationCount)
	}

	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("Synchronization completed successfully in %s!", elapsed.Round(time.Millisecond))
}
