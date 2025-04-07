package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

// Get environment variable by name. If it does not exist, return default value.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

const (
	WAKEUP_USER     string = "WAKEUP_USER"
	WAKEUP_PASSWORD string = "WAKEUP_PASSWORD"
	WAKEUP_SERVER   string = "WAKEUP_SERVER"
	WAKEUP_INSTANCE string = "WAKEUP_INSTANCE"
	WAKEUP_DATABASE string = "WAKEUP_DATABASE"
	WAKEUP_PORT     string = "WAKEUP_PORT"
	WAKEUP_DSN      string = "WAKEUP_DSN"
)

// Build connection string for Azure SQL Database from environment variables.
// Uses: WAKEUP_DSN, WAKEUP_APP_NAME, WAKEUP_DATABASE, WAKEUP_SERVER, WAKEUP_PORT, WAKEUP_USER, WAKEUP_PASSWORD, WAKEUP_INSTANCE
func buildConnectionString(
	server string,
	port string,
	instance string,
	database string,
	username string,
	password string,
	dsn string,
) string {
	if dsn != "" {
		return dsn
	}

	q := url.Values{}
	q.Add("AppName", "ghcr.io/redmer/azure-wakeup-db")
	q.Add("DisableRetry", fmt.Sprintf("%t", false))

	timeout := time.Duration(5) * time.Minute // 5 min timeout
	q.Add("DialTimeout", strconv.FormatFloat(float64(timeout/time.Second), 'f', 0, 64))

	if database != "" {
		q.Add("database", database)
	}

	res := url.URL{
		Scheme: "sqlserver",
		Host:   fmt.Sprintf("%s:%s", server, port),
		User:   url.UserPassword(username, password),
	}

	if instance != "" {
		res.Path = instance
	}

	if len(q) > 0 {
		res.RawQuery = q.Encode()
	}

	return res.String()
}

// isThrottlingError checks if the error is Azure SQL throttling error (40613)
func isThrottlingError(err error) bool {
	if err == nil {
		return false
	}

	// Check if the error contains "40613"
	return strings.Contains(err.Error(), "40613")
}

func addJitter(delay time.Duration) time.Duration {
	jitter := time.Duration(rand.Float64() * float64(delay) * 0.1) // 10% jitter
	return delay + jitter
}

// Continuously retry with exponential backoff until the maximum number of retries is reached.
func retryWithThrottlingError(ctx context.Context, connString string) (*sql.DB, error) {
	maxRetries := 6
	// 1: wait 0 sec
	// 2: wait 12 sec = cumulatively 12
	// 3: wait 24 sec = cumulatively 36
	// 4: wait 48 sec = cumulatively 84
	// 5: wait 96 sec = cumulatively 145
	// 6: wait 192 sec = cumulatively 237
	retryDelay := time.Duration(12) * time.Second
	var lastErr error

	for attempt := range maxRetries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if attempt > 0 {
				log.Printf("Retry attempt %d/%d after %v delay", attempt+1, maxRetries, retryDelay)
				time.Sleep(addJitter(retryDelay))
			}

			db, err := connectWithString(connString)
			if err == nil { // success
				return db, nil
			}

			lastErr = err
			if !isThrottlingError(err) { // not throttling error
				return nil, err
			}

			retryDelay *= 2
		}
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %v", maxRetries, lastErr)
}

// Return a working sql.DB connection based on a connection string
func connectWithString(connString string) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(6 * time.Minute)

	// Test connection with context, 5 minute timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	return db, nil
}

// Ensure a connection with an Azure DB that may be auto-paused.
// Wait and try for 5 minutes to wake it up.
func main() {
	server := flag.String("server", os.Getenv(WAKEUP_SERVER), "Database server")
	port := flag.String("port", getEnv(WAKEUP_PORT, "1433"), "Database port")
	instance := flag.String("instance", os.Getenv(WAKEUP_INSTANCE), "SQL Server instance name")
	database := flag.String("database", os.Getenv(WAKEUP_DATABASE), "Database name")
	user := flag.String("user", os.Getenv(WAKEUP_USER), "Database user")
	password := flag.String("password", os.Getenv(WAKEUP_PASSWORD), "Database password")
	dsn := flag.String("dsn", os.Getenv(WAKEUP_DSN), "Database connection string")
	help := flag.Bool("help", false, "Show this help message")
	verbose := flag.Bool("verbose", false, "Verbose output")

	flag.Parse()

	if *help {
		why := `Connect to awaken a paused Azure DB.

  Provide connection details or environment variables to connect. For all options,
  there is a corresponding environment variable named WAKEUP_<option_name>.

	--server=myserver -> WAKEUP_SERVER=myserver`

		fmt.Println(why)
		flag.Usage()

		os.Exit(0)
	}

	connectionString := *dsn
	// If no DSN provided, try to build from environment variables
	if connectionString == "" {
		connectionString = buildConnectionString(*server, *port, *instance, *database, *user, *password, *dsn)
	}

	if *verbose {
		fmt.Printf("Connecting with '%v'.", connectionString)
	}

	if connectionString == "" || strings.HasPrefix(connectionString, "sqlserver://:@:1433?") {
		fmt.Println("Error: no connection string provided via --dsn flag or environment variables")
		os.Exit(1)
	}

	// Actually connect to the database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	conn, err := retryWithThrottlingError(ctx, connectionString)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Connection successful: database is awake.")
}
