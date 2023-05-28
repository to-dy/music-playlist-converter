package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/to-dy/music-playlist-converter/api/router/routes/auth"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")

	auth.SetupAuthRoutes(api)
}
