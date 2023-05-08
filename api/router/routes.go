package router

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"

	"github.com/to-dy/music-playlist-converter/api/services/spotify"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	api.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Get("/login_test", func(c *fiber.Ctx) error {
		state := utils.UUIDv4()

		c.Cookie(&fiber.Cookie{
			Name:  "spotify_auth_state",
			Value: state,
		})

		scopes := "playlist-modify-public playlist-read-private"

		// create query string to append to redirect url
		queryString := "client_id=" + os.Getenv("SPOTIFY_CLIENT_ID") + "&response_type=code&redirect_uri=" + os.Getenv("SPOTIFY_REDIRECT_URI") + "&scope=" + scopes + "&state=" + state

		return c.Redirect("https://accounts.spotify.com/authorize?" + queryString)

	})

	api.Get("/spotify_callback", func(c *fiber.Ctx) error {
		code := c.Query("code")
		state := c.Query("state")
		v_error := c.Query("error")
		storedState := c.Cookies("spotify_auth_state")

		if v_error != "" {
			return c.Redirect("/#?error=" + v_error)
		} else if state == "" || state != storedState {
			return c.Redirect("/#error")
		} else {
			c.ClearCookie("spotify_auth_state")

			spotify.FetchAccessToken(code)

			return nil

		}

	})

}
