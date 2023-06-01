package router

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/to-dy/music-playlist-converter/api/router/routes"
)

var SessionStore *session.Store

func SetupServer() {
	app := fiber.New()

	// server logging
	app.Use(logger.New())

	// recover from panic
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))

	routes.SetupRoutes(app)

	if err := app.Listen(":" + os.Getenv("PORT")); err != nil {
		log.Fatal(err)
	}

}
