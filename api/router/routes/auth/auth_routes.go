package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"golang.org/x/oauth2"

	"github.com/to-dy/music-playlist-converter/api/handlers"
	"github.com/to-dy/music-playlist-converter/api/services/spotify"
	"github.com/to-dy/music-playlist-converter/api/services/youtube"
)

func SetupAuthRoutes(app *fiber.App) {
	authRoutes := app.Group("/auth")

	authRoutes.Get("/spotify", handlers.InitiateOAuthFlow)

	authRoutes.Get("/spotify_callback", func(c *fiber.Ctx) error {
		code := c.Query("code")
		state := c.Query("state")
		v_error := c.Query("error")
		storedState := c.Cookies("spotify_auth_state")

		if v_error != "" {
			return c.Redirect("/#?error=" + v_error)
		}

		if state == "" || state != storedState {
			return c.Redirect("/#error")
		}

		c.ClearCookie("spotify_auth_state")

		token, err := spotify.OauthConfig.Exchange(c.Context(), code)

		if err != nil {
			// Handle error
			return err
		}

		spotify.StoreOauthToken(token)

		return c.SendStatus(fiber.StatusOK)

	})

	authRoutes.Get("/youtube", func(c *fiber.Ctx) error {
		state := utils.UUIDv4()

		c.Cookie(&fiber.Cookie{
			Name:  "youtube_auth_state",
			Value: state,
		})

		url := youtube.OauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

		return c.Redirect(url)
	})

	authRoutes.Get("/youtube_callback", func(c *fiber.Ctx) error {
		code := c.Query("code")
		state := c.Query("state")
		authError := c.Query("error")
		storedState := c.Cookies("youtube_auth_state")

		if authError != "" {
			return c.Redirect("/#?error=" + authError)
		}

		if state == "" || state != storedState {
			return c.Redirect("/#error=wrong-state")
		}

		c.ClearCookie("youtube_auth_state")

		token, err := spotify.OauthConfig.Exchange(c.Context(), code)

		if err != nil {
			// Handle error
			return err
		}

		youtube.StoreOauthToken(token)

		return c.SendStatus(fiber.StatusOK)

	})

}
