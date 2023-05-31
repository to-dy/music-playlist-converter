package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/to-dy/music-playlist-converter/api/router/routes/auth"
	"github.com/to-dy/music-playlist-converter/api/router/routes/playlist"
)

func SetupRoutes(app *fiber.App) {
	apiRoutes := app.Group("/api")

	auth.SetupAuthRoutes(apiRoutes)
	playlist.SetupPlaylistRoutes(apiRoutes)
}
