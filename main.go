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
		flag.Usage()
		os.Exit(0)
	}

	// Example 1: Using connection string
	conn, err := connectWithString(*dsn)
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Successfully connected to database!")
}
