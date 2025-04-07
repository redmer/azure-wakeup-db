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

// Get environment variable by name. If it does not exist, return a default value.
func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Build connection string for Azure SQL Database from environment variables.
func BuildDSN(
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

// If error provided is an Azure SQL throttling error (40613), also caused by paused instances.
func isThrottlingError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "40613")
}

// Add 10% timing jitter to a time.Duration
func addJitter(delay time.Duration) time.Duration {
	jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
	return delay + jitter
}

// With pauses, Retry to connect until the maximum number of retries is reached.
func ThrottledRetry[T any](ctx context.Context, connectFunc func() (T, error)) (T, error) {
	var zeroValue T
	maxRetries := 15
	retryDelay := time.Duration(25) * time.Second
	var lastErr error

	for attempt := range maxRetries {
		select {
		case <-ctx.Done():
			return zeroValue, ctx.Err()
		default:
			if attempt > 0 {
				log.Printf("attempt %d/%d after %v delay", attempt+1, maxRetries, retryDelay)
				time.Sleep(addJitter(retryDelay))
			} else {
				log.Printf("attempt 1/%d", maxRetries)
			}

			result, err := connectFunc()
			if err == nil { // success
				return result, nil
			}

			lastErr = err
			if !isThrottlingError(err) { // not throttling error
				return zeroValue, err
			}
		}
	}

	return zeroValue, fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

// Return a working sql.DB connection based on a connection string
func ConnectAndPing(connString string) (*sql.DB, error) {
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

const (
	WAKEUP_USER     string = "WAKEUP_USER"
	WAKEUP_PASSWORD string = "WAKEUP_PASSWORD"
	WAKEUP_SERVER   string = "WAKEUP_SERVER"
	WAKEUP_INSTANCE string = "WAKEUP_INSTANCE"
	WAKEUP_DATABASE string = "WAKEUP_DATABASE"
	WAKEUP_PORT     string = "WAKEUP_PORT"
	WAKEUP_DSN      string = "WAKEUP_DSN"
)

// Ensure a connection with an Azure DB that may be auto-paused.
// Wait and try for 5 minutes to wake it up.
func main() {
	server := flag.String("server", os.Getenv(WAKEUP_SERVER), "Database server")
	port := flag.String("port", GetEnv(WAKEUP_PORT, "1433"), "Database port")
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

  Provide connection details to connect. All environment variable name start with WAKEUP_<option_name>.
  Command line options passed have higher priority. The DSN option always overrides any and all other values.`

		fmt.Println(why)
		flag.Usage()

		os.Exit(0)
	}

	connectionString := *dsn
	// If no DSN provided, try to build from environment variables
	if connectionString == "" {
		connectionString = BuildDSN(*server, *port, *instance, *database, *user, *password, *dsn)
	}

	if *verbose {
		log.Printf("Connecting with '%v'.\n", connectionString)
	}

	if connectionString == "" || strings.HasPrefix(connectionString, "sqlserver://:@:1433?") {
		log.Fatal("Error: no connection string provided via --dsn flag or environment variables")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Actually connect to the database
	conn, err := ThrottledRetry(ctx,
		func() (*sql.DB, error) {
			return ConnectAndPing(connectionString)
		})
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	log.Println("Connection successful: database is awake.")
}
