package router

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// var port string = os.Getenv("PORT")

func SetupServer() {
	app := fiber.New()

	// server logging
	app.Use(logger.New())

	SetupRoutes(app)

	err := app.Listen(fmt.Sprintf(":%s", os.Getenv("PORT")))

	if err != nil {
		log.Fatal(err)
	}

}
