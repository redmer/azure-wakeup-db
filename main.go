package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

type DBConnection struct {
	db *sql.DB
}

type ConnectionConfig struct {
	Server       string
	Port         int
	Database     string
	User         string
	Password     string
	InstanceName string
	Encrypt      bool
	TrustCert    bool
	ConnTimeout  int
	ReadTimeout  int
	WriteTimeout int
}

func NewDBConnection(config interface{}) (*DBConnection, error) {
	var db *sql.DB
	var err error

	switch v := config.(type) {
	case string:
		db, err = connectWithString(v)
	case map[string]string:
		db, err = connectWithParams(v)
	case *ConnectionConfig:
		db, err = connectWithConfig(v)
	default:
		return nil, fmt.Errorf("unsupported configuration type")
	}

	if err != nil {
		return nil, err
	}

	return &DBConnection{db: db}, nil
}

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

func connectWithParams(params map[string]string) (*sql.DB, error) {
	var connParts []string
	for key, value := range params {
		connParts = append(connParts, fmt.Sprintf("%s=%s", key, value))
	}
	connString := strings.Join(connParts, ";")

	return connectWithString(connString)
}

func connectWithConfig(config *ConnectionConfig) (*sql.DB, error) {
	params := map[string]string{
		"server":   config.Server,
		"database": config.Database,
		"user id":  config.User,
		"password": config.Password,
	}

	if config.Port > 0 {
		params["port"] = fmt.Sprintf("%d", config.Port)
	}
	if config.InstanceName != "" {
		params["instance name"] = config.InstanceName
	}
	if config.Encrypt {
		params["encrypt"] = "true"
	}
	if config.TrustCert {
		params["trustservercertificate"] = "true"
	}
	if config.ConnTimeout > 0 {
		params["connection timeout"] = fmt.Sprintf("%d", config.ConnTimeout)
	}
	if config.ReadTimeout > 0 {
		params["read timeout"] = fmt.Sprintf("%d", config.ReadTimeout)
	}
	if config.WriteTimeout > 0 {
		params["write timeout"] = fmt.Sprintf("%d", config.WriteTimeout)
	}

	return connectWithParams(params)
}

func (dc *DBConnection) Close() error {
	if dc.db != nil {
		return dc.db.Close()
	}
	return nil
}

// Example query method
func (dc *DBConnection) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return dc.db.QueryContext(ctx, query, args...)
}

func main() {
	// Example 1: Using connection string
	connString := "server=your-server.database.windows.net;user id=your-username;password=your-password;database=your-database"
	db1, err := NewDBConnection(connString)
	if err != nil {
		fmt.Printf("Error connecting with string: %v\n", err)
		return
	}
	defer db1.Close()

	// Example 2: Using connection parameters map
	connParams := map[string]string{
		"server":   "your-server.database.windows.net",
		"user id":  "your-username",
		"password": "your-password",
		"database": "your-database",
	}
	db2, err := NewDBConnection(connParams)
	if err != nil {
		fmt.Printf("Error connecting with params: %v\n", err)
		return
	}
	defer db2.Close()

	// Example 3: Using connection config struct
	config := &ConnectionConfig{
		Server:       "your-server.database.windows.net",
		Database:     "your-database",
		User:         "your-username",
		Password:     "your-password",
		Port:         1433,
		Encrypt:      true,
		TrustCert:    true,
		ConnTimeout:  30,
		ReadTimeout:  30,
		WriteTimeout: 30,
	}
	db3, err := NewDBConnection(config)
	if err != nil {
		fmt.Printf("Error connecting with config: %v\n", err)
		return
	}
	defer db3.Close()

	// Example query
	ctx := context.Background()
	rows, err := db3.Query(ctx, "SELECT * FROM YourTable")
	if err != nil {
		fmt.Printf("Error querying database: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("Successfully connected to database!")
}
