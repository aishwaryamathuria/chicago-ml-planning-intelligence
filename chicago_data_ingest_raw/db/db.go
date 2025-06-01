package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var connection *sql.DB

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("❌ Error loading .env file")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	connection, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Failed to connect to DB: %v", err)
	}

	err = connection.Ping()
	if err != nil {
		log.Fatalf("❌ Ping error: %v", err)
	}
}

// GetDB returns standard sql.DB for INSERT-based usage
func GetDB() *sql.DB {
	return connection
}

// GetPgxConn returns a pgx.Conn for COPY-based usage
func GetPgxConn() *pgx.Conn {
	ctx := context.Background()
	pgxConnStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	conn, err := pgx.Connect(ctx, pgxConnStr)
	if err != nil {
		log.Fatalf("❌ Failed to connect using pgx: %v", err)
	}
	return conn
}
