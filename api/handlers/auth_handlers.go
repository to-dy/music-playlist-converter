package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/to-dy/music-playlist-converter/api/services/spotify"
)

func InitiateOAuthFlow(c *fiber.Ctx) error {
	path := c.Request().URI().Path()

	state := utils.UUIDv4()

	c.Cookie(&fiber.Cookie{
		Name:  "spotify_auth_state",
		Value: state,
	})

	url := spotify.OauthConfig.AuthCodeURL(state)

	return c.Redirect(url)

}
