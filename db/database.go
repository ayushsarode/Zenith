package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

func InitDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Read DB credentials from environment variables
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	// Format the connection string
	connStr := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		dbUser, dbPassword, dbName, dbHost, dbPort, dbSSLMode,
	)

	db, err = sql.Open("postgres", connStr)

	if err != nil {
		log.Fatalf("Failed to connect to databse:", err)
	}

	createTables()
}

func createTables() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL
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
		log.Fatal("Failed to create tables:", err)
	}
}