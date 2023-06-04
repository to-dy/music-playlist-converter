package handlers

import (
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"golang.org/x/oauth2"

	"github.com/to-dy/music-playlist-converter/api/services/spotify"
	"github.com/to-dy/music-playlist-converter/api/services/youtube"
	"github.com/to-dy/music-playlist-converter/api/stores/session"
)

const (
	SPOTIFY_AUTH_STATE = "spotify_auth_state"
	YOUTUBE_AUTH_STATE = "youtube_auth_state"
)

func InitiateOAuthFlow(c *fiber.Ctx) error {
	path := c.Path()
	path = strings.TrimPrefix(path, "/api/auth")

	state := utils.UUIDv4()

	sess, err := session.Store.Get(c)
	if err != nil {
		log.Println("Error getting session - " + err.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	switch path {
	case "/spotify":
		c.Cookie(&fiber.Cookie{
			Name:  SPOTIFY_AUTH_STATE,
			Value: state,
		})

		url := spotify.OauthConfig.AuthCodeURL(state)

		sess.Set(session.ConvertTo, "spotify")
		sess.Save()

		return c.Redirect(url, fiber.StatusTemporaryRedirect)

	case "/youtube":
		c.Cookie(&fiber.Cookie{
			Name:  YOUTUBE_AUTH_STATE,
			Value: state,
		})

		url := youtube.OauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

		sess.Set(session.ConvertTo, "youtube")
		sess.Save()

		return c.Redirect(url, fiber.StatusTemporaryRedirect)

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

	sess, err := session.Store.Get(c)
	if err != nil {
		log.Println("Error getting session - " + err.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	switch path {

	case "/spotify_callback":
		storedState := c.Cookies(SPOTIFY_AUTH_STATE)

		if authError != "" {
			return c.Redirect(os.Getenv("UI_BASE_URL") + "/auth?error=" + authError)
		}

		if state == "" || state != storedState {
			return c.Redirect(os.Getenv("UI_BASE_URL") + "/auth?error=state-mismatch")
		}

		c.ClearCookie(SPOTIFY_AUTH_STATE)

		token, err := spotify.OauthConfig.Exchange(c.Context(), code)

		if err != nil {
			return c.Redirect(os.Getenv("UI_BASE_URL") + "/auth?error=" + err.Error())
		}

		spotify.StoreAuthCodeToken(token, sess.ID())

		sess.Set(session.AuthCodeToken, token.AccessToken)
		sess.Save()

		return c.Redirect(os.Getenv("UI_BASE_URL") + "/auth?success=true")

	case "/youtube_callback":
		storedState := c.Cookies(YOUTUBE_AUTH_STATE)

		if authError != "" {
			return c.Redirect(os.Getenv("UI_BASE_URL") + "/auth?error=" + authError)
		}

		if state == "" || state != storedState {
			return c.Redirect(os.Getenv("UI_BASE_URL") + "/auth?error=state-mismatch")
		}

		c.ClearCookie(YOUTUBE_AUTH_STATE)

		token, err := youtube.OauthConfig.Exchange(c.Context(), code)

		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		youtube.StoreAuthCodeToken(token, sess.ID())

		sess.Set(session.AuthCodeToken, token.AccessToken)
		sess.Save()

		return c.Redirect(os.Getenv("UI_BASE_URL") + "/auth?success=true")

	default:
		return c.Redirect(os.Getenv("UI_BASE_URL") + "/auth?error=unsupported-callback")
	}
}
