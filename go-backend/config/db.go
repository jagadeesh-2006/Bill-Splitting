package config

import(
	"fmt"
	"os"
	"github.com/jackc/pgx/v5/pgxpool"  
	"log"
	"context"
)

var DB *pgxpool.DB

func ConnectDB() {
	DB_URL := os.Getenv("DATABASE_URL")
	if(DB_URL == "") {
		log.Fatal("db url is not found")
	}
	pool , err := pgxpool.New(context.Background(), DB_URL)  
	if(err != nil){
		log.Fatal("unable to create a connection pool")
	}
	//check connection to db by sending ping request
	err = pool.Ping(context.Background())  
	if(err != nil){
		log.Fatal("unable to connect to the database")
	}
	
	DB = pool 
	fmt.Println("connected to the database successfully")

}