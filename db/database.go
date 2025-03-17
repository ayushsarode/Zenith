package db

import (
    "database/sql"
    "fmt"

    "os"
    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
)

var db *sql.DB


func InitDB() (*sql.DB, error) {
    err := godotenv.Load()
    if err != nil {
        return nil, fmt.Errorf("error loading .env file: %v", err)
    }
    
    dbUser := os.Getenv("DB_USER")
    dbPassword := os.Getenv("DB_PASSWORD")
    dbName := os.Getenv("DB_NAME")
    dbHost := os.Getenv("DB_HOST")
    dbPort := os.Getenv("DB_PORT")
    dbSSLMode := os.Getenv("DB_SSLMODE")
    
    connStr := fmt.Sprintf(
        "user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
        dbUser, dbPassword, dbName, dbHost, dbPort, dbSSLMode,
    )
    
    database, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %v", err)
    }
    
    if err := database.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %v", err)
    }
    
    // Store in package variable for backwards compatibility
    db = database
    
    if err := createTables(database); err != nil {
        return nil, err
    }
    
    return database, nil
}

func createTables(db *sql.DB) error {
    query := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        username TEXT UNIQUE NOT NULL,
        password TEXT NOT NULL
    );
    CREATE TABLE IF NOT EXISTS rooms (
        id SERIAL PRIMARY KEY,
        name TEXT UNIQUE NOT NULL
    );
    CREATE TABLE IF NOT EXISTS messages (
        id SERIAL PRIMARY KEY,
        user_id INTEGER REFERENCES users(id),
        room_id INTEGER REFERENCES rooms(id),
        message TEXT NOT NULL
    );
    `
    
    _, err := db.Exec(query)
    if err != nil {
        return fmt.Errorf("failed to create tables: %v", err)
    }
    
    return nil
}

// For backwards compatibility - use these if you're not using the returned DB connection

func QueryRow(query string, args ...interface{}) *sql.Row {
    return db.QueryRow(query, args...)
}

func Exec(query string, args ...interface{}) (sql.Result, error) {
    return db.Exec(query, args...)
}

func Query(query string, args ...interface{}) (*sql.Rows, error) {
    return db.Query(query, args...)
}