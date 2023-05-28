package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"golang.org/x/oauth2"

	"github.com/to-dy/music-playlist-converter/api/services/spotify"
	"github.com/to-dy/music-playlist-converter/api/services/youtube"
)

const (
	SPOTIFY_AUTH_STATE = "spotify_auth_state"
	YOUTUBE_AUTH_STATE = "youtube_auth_state"
)

func InitiateOAuthFlow(c *fiber.Ctx) error {
	path := c.Path()
	path = strings.TrimPrefix(path, "/api/auth")

	state := utils.UUIDv4()

	switch path {

	case "/spotify":
		c.Cookie(&fiber.Cookie{
			Name:  SPOTIFY_AUTH_STATE,
			Value: state,
		})

		url := spotify.OauthConfig().AuthCodeURL(state)

		return c.Redirect(url)

	case "/youtube":
		c.Cookie(&fiber.Cookie{
			Name:  YOUTUBE_AUTH_STATE,
			Value: state,
		})

		url := youtube.OauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

		return c.Redirect(url)

	default:
		return c.SendStatus(fiber.StatusNotFound)
	}

}

func HandleOAuthCallback(c *fiber.Ctx) error {
	path := c.Path()
	path = strings.TrimPrefix(path, "/api/auth")

	code := c.Query("code")
	state := c.Query("state")
	authError := c.Query("error")

	switch path {

	case "/spotify_callback":
		storedState := c.Cookies(SPOTIFY_AUTH_STATE)

		if authError != "" {
			return c.Status(fiber.StatusBadRequest).SendString(authError)
		}

		if state == "" || state != storedState {
			return c.Status(fiber.StatusBadRequest).SendString("state-mismatch")
		}

		c.ClearCookie(SPOTIFY_AUTH_STATE)

		token, err := spotify.OauthConfig().Exchange(c.Context(), code)

		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		spotify.StoreOauthToken(token)

		return c.SendStatus(fiber.StatusOK)

	case "/youtube_callback":
		storedState := c.Cookies(YOUTUBE_AUTH_STATE)

		if authError != "" {
			return c.Redirect("/#?error=" + authError)
		}

		if state == "" || state != storedState {
			return c.Redirect("/#error=state-mismatch")
		}

		c.ClearCookie(YOUTUBE_AUTH_STATE)

		token, err := youtube.OauthConfig.Exchange(c.Context(), code)

		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		youtube.StoreOauthToken(token)

		return c.SendStatus(fiber.StatusOK)

	default:
		return c.SendStatus(fiber.StatusNotFound)
	}

}

func HandleOAuthError(c *fiber.Ctx) {
}

// func CompleteAuthentications(c *fiber.Ctx) {
// }
