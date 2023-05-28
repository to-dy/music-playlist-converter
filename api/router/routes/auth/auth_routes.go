package auth

import (
	"github.com/gofiber/fiber/v2"

	"github.com/to-dy/music-playlist-converter/api/handlers"
)

func SetupAuthRoutes(router fiber.Router) {

	authRouter := router.Group("/auth")

	authRouter.Get("/spotify", handlers.InitiateOAuthFlow)

	authRouter.Get("/spotify_callback", handlers.HandleOAuthCallback)

	authRouter.Get("/youtube", handlers.InitiateOAuthFlow)

	authRouter.Get("/youtube_callback", handlers.HandleOAuthCallback)
}
