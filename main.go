package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
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
	database string,
	instance string,
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
	// 5 min timeout
	q.Add("DialTimeout", strconv.FormatFloat(float64((time.Duration(5)*time.Minute)/time.Second), 'f', 0, 64))

	if database != "" {
		q.Add("database", database)
	}

	host := fmt.Sprintf("%s:%s", server, port)

	res := url.URL{
		Scheme: "sqlserver",
		Host:   host,
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

type Options struct {
	name   string
	envvar string
	help   string
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

	if connectionString == "" || strings.HasPrefix(connectionString, "sqlserver://:@:1433?") {
		fmt.Println("Error: No connection string provided via --dsn flag or environment variables")
		os.Exit(1)
	}

	// Actually connect to the database
	conn, err := connectWithString(connectionString)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Connection successful: database is awake.")
}
