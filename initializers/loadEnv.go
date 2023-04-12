package initializers

import (
	"log"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load("internal/env/.env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}
}
