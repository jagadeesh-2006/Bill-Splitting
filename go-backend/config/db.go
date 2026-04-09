package config

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func ConnectDB() {
	DB_URL := os.Getenv("DATABASE_URL")
	if DB_URL == "" {
		log.Fatal("db url is not found")
	}
	// pgxpool.New creates a new connection pool to the database using the provided URL. It returns a pointer to the pool and an error if any occurs during the creation of the pool.
	pool, err := pgxpool.New(context.Background(), DB_URL)
	if err != nil {
		log.Fatal("unable to create a connection pool")
	}
	//check connection to db by sending ping request
	err = pool.Ping(context.Background())
	if err != nil {
		log.Fatal("unable to connect to the database")
	}

	DB = pool
	fmt.Println("connected to the database successfully")

}
