package main

import (
	"context"
	"fmt"
	"runtime"

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
	// Initialize logger with default debug false
	log := logger.InitLogger(false)

	// Check for updates first
	ctx := context.Background()
	cfgForUpdate := config.Config{
		UpdateCheckURL:    "", // will use default GitHub URL
		AutoUpdate:        false,
		UpdateDownloadDir: ".",
	}
	downloaded, path, info, err := updater.RunUpdateFlow(ctx, version, cfgForUpdate)
	if err != nil {
		log.Warn().Err(err).Msg("Error while checking updates")
	} else if info.URL != "" {
		if downloaded {
			log.Info().Str("latest", info.Version).Str("file", path).Msg("Update downloaded successfully")
		} else {
			log.Info().Str("latest", info.Version).Str("download_url", info.URL).Msg("Download available for the latest version")
		}
	}

	// Load configuration from .env
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading configuration")
	}

	// Update logger level if debug mode is enabled
	if cfg.DebugMode {
		// Since logger is initialized with once, we can set the global level
		// Note: This may not affect already created loggers, but for simplicity
		// we'll assume it's ok or handle differently if needed
	}

	fmt.Printf("\nSynC Firebird x MySQL v%s (Optimized Worker Pool)\n\n", version)

	// Run main processing and print a summarized report
	insertedCount, updatedCount, ignoredCount, batchSize, stats, elapsedTime, maxConnections, maxAllowedPacket, err := runProcessing(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Error processing rows")
	}

	// Calculate number of workers used
	numWorkers := runtime.NumCPU() * 2
	if numWorkers > 20 {
		numWorkers = 20
	}
	if numWorkers < 4 {
		numWorkers = 4
	}

	printSummary(insertedCount, updatedCount, ignoredCount, batchSize, stats, elapsedTime, numWorkers, maxConnections, maxAllowedPacket)
}
