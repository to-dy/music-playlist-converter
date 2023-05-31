package playlist

import (
	"github.com/gofiber/fiber/v2"

	"github.com/to-dy/music-playlist-converter/api/handlers"
)

func SetupPlaylistRoutes(router fiber.Router) {

	playlistRouter := router.Group("/playlist")

	playlistRouter.Get("/verify", handlers.VerifyPlaylist)

}
