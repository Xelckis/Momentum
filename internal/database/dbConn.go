package database

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"os"
)

var conn *pgxpool.Pool

func ConnectDB() *pgxpool.Pool {
	var err error
	conn, err = pgxpool.New(context.Background(), os.Getenv("MOMENTUMDB"))
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		conn.Close()
		log.Fatalf("Unable to ping database: %v\n", err)
	}

	return conn

}
