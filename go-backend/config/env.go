package config

import(
	"log"
	"github.com/joho/godotenv"
)

func loadenv(){
	err := godotenv.Load()   
	if err != nil {
		log.Fatal("Error loading .env file")
	}else {
		 log.Println("env file loaded succesufully")
	}
}