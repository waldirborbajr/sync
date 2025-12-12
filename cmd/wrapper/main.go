//go:build wrapper
// +build wrapper

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/waldirborbajr/sync/config"
	"github.com/waldirborbajr/sync/db"
	"github.com/waldirborbajr/sync/logger"
)

func main() {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		printConfigError(err)
		os.Exit(1)
	}

	// Init logger
	log := logger.InitLogger(cfg.DebugMode)

	// Check Firebird
	if err := checkFirebird(cfg); err != nil {
		log.Error().Err(err).Msg("Firebird connection check failed")
		printFirebirdTips(err)
		os.Exit(2)
	}
	log.Info().Msg("Firebird check OK")

	// Check MySQL
	if err := checkMySQL(cfg); err != nil {
		log.Error().Err(err).Msg("MySQL connection check failed")
		printMySQLTips(err)
		os.Exit(3)
	}
	log.Info().Msg("MySQL check OK")

	// Optionally, check statements
	if err := checkMySQLStatements(cfg); err != nil {
		log.Warn().Err(err).Msg("MySQL statements check failed")
		fmt.Fprintf(os.Stderr, "Warning: could not prepare test statements: %v\n", err)
		os.Exit(4)
	}

	log.Info().Msg("All startup checks passed. You're good to run the sync application.")
	fmt.Println("All startup checks passed. No issues detected.")
}

func printConfigError(err error) {
	fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
	if strings.Contains(err.Error(), "missing required MySQL") {
		fmt.Fprintln(os.Stderr, "Suggested action: ensure MYSQL_USER, MYSQL_PASSWORD, MYSQL_HOST, MYSQL_PORT and MYSQL_DATABASE are set in .env or environment")
	}
	if strings.Contains(err.Error(), "missing required Firebird") {
		fmt.Fprintln(os.Stderr, "Suggested action: ensure FIREBIRD_USER, FIREBIRD_PASSWORD, FIREBIRD_HOST, and FIREBIRD_PATH are set in .env or environment")
	}
	fmt.Fprintln(os.Stderr, "Tip: run with DEBUG_MODE=true for more detailed logs.")
}

func checkFirebird(cfg config.Config) error {
	// Attempt to connect
	fb, err := db.ConnectFirebird(cfg)
	if err != nil {
		return err
	}
	defer fb.Close()
	return nil
}

func printFirebirdTips(err error) {
	fmt.Fprintf(os.Stderr, "Firebird check error: %v\n", err)
	fmt.Fprintln(os.Stderr, "Suggested actions:")
	fmt.Fprintln(os.Stderr, " - Verify Firebird service is running and reachable from this host (default port 3050)")
	fmt.Fprintln(os.Stderr, " - Verify credentials in .env: FIREBIRD_USER, FIREBIRD_PASSWORD, FIREBIRD_HOST, FIREBIRD_PATH")
	fmt.Fprintln(os.Stderr, " - Try connecting using isql or a Firebird client to the same host and DSN")
	fmt.Fprintln(os.Stderr, " - If using a Docker container, ensure networking allows the connection")
}

func checkMySQL(cfg config.Config) error {
	my, err := db.ConnectMySQL(cfg)
	if err != nil {
		return err
	}
	defer my.Close()
	return nil
}

func printMySQLTips(err error) {
	fmt.Fprintf(os.Stderr, "MySQL check error: %v\n", err)
	fmt.Fprintln(os.Stderr, "Suggested actions:")
	fmt.Fprintln(os.Stderr, " - Verify MySQL service is running and reachable on the configured host and port")
	fmt.Fprintln(os.Stderr, " - Verify credentials: MYSQL_USER, MYSQL_PASSWORD, MYSQL_HOST, MYSQL_PORT, MYSQL_DATABASE")
	fmt.Fprintln(os.Stderr, " - Ensure the MySQL user has permissions to connect and run queries")
	fmt.Fprintln(os.Stderr, " - Check that the server allows your IP and that security groups/firewalls are configured")
	fmt.Fprintln(os.Stderr, " - For more details, set DEBUG_MODE=true and re-run the checks to get detailed logs")
}

func checkMySQLStatements(cfg config.Config) error {
	my, err := db.ConnectMySQL(cfg)
	if err != nil {
		return err
	}
	defer my.Close()
	_, _, err = db.PrepareStatements(my)
	return err
}
