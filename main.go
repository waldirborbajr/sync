package main

import (
	"context"
	"fmt"

	"github.com/waldirborbajr/sync/config"
	"github.com/waldirborbajr/sync/logger"
	"github.com/waldirborbajr/sync/updater"
)

// version is set at build time using -ldflags="-X main.version=VERSION"
var version string

// ANSI color codes
const (
	redBold   = "\033[1;31m"
	greenBold = "\033[1;32m"
	reset     = "\033[0m"
)

func main() {
	// Load configuration from .env
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Error loading configuration")
	}

	// Initialize logger
	log := logger.InitLogger(cfg.DebugMode)

	fmt.Printf("\nSynC Firebird x MySQL v%s\n\n", version)

	// Check for updates if configured
	ctx := context.Background()
	downloaded, path, info, err := updater.RunUpdateFlow(ctx, version, cfg)
	if err != nil {
		log.Warn().Err(err).Msg("Error while checking/downloading updates")
	} else if info.URL != "" {
		if downloaded {
			log.Info().Str("latest", info.Version).Str("file", path).Msg("Update downloaded successfully")
		} else {
			log.Info().Str("latest", info.Version).Str("download_url", info.URL).Msg("Download available for the latest version")
		}
	}

	// Run main processing and print a summarized report
	insertedCount, updatedCount, ignoredCount, batchSize, stats, elapsedTime, maxConnections, maxAllowedPacket, err := runProcessing(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Error processing rows")
	}

	semaphoreSize := int(float64(maxConnections) * 0.75)
	if semaphoreSize < 10 {
		semaphoreSize = 10
	} else if semaphoreSize > 100 {
		semaphoreSize = 100
	}

	printSummary(insertedCount, updatedCount, ignoredCount, batchSize, stats, elapsedTime, semaphoreSize, maxConnections, maxAllowedPacket)
}
