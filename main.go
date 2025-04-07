package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

func connectWithString(connString string) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection with context, 10 sec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	return db, nil
}

func main() {
	dsn := flag.String("dsn", "", "Database connection string")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help || *dsn == "" {
		examples := `
Connection string examples:
  --dsn 'sqlserver://username:password@host/instance'
  --dsn 'server=localhost;user id=sa;database=master;app name=MyAppName'
  --dsn 'odbc:server=localhost;user id=sa;password={foo;bar}'

  All variants described at <https://github.com/microsoft/go-mssqldb/blob/main/README.md#the-connection-string-can-be-specified-in-one-of-three-formats>")`
		flag.Usage()
		fmt.Println(examples)
		os.Exit(0)
	}

	// Example 1: Using connection string
	conn, err := connectWithString(*dsn)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Successfully connected to database!")
}
