package handlers

import (
	"log"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gookit/goutil/arrutil"

	"github.com/to-dy/music-playlist-converter/api/services/spotify"
	"github.com/to-dy/music-playlist-converter/api/services/youtube"
)

var supportedPlaylistsHost = []string{"music.youtube.com", "www.youtube.com", "youtube.com", "open.spotify.com"}

func VerifyPlaylist(c *fiber.Ctx) error {
	urlQ := c.Query("url")

	if urlQ == "" {
		log.Println("url query parameter is empty")

		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Errors: Errors{getBadRequestError("The `url` query parameter is required.")},
		})
	}

	parsedURL, urlParseErr := url.Parse(urlQ)

	if urlParseErr != nil {
		log.Println("Error parsing url - " + urlQ + " error - " + urlParseErr.Error())

		return c.SendStatus(fiber.StatusInternalServerError)
	}

	if !arrutil.Contains(supportedPlaylistsHost, parsedURL.Host) {
		log.Println("unsupported playlist host - " + parsedURL.Host)

		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Errors: Errors{getBadRequestError("Invalid or unsupported playlist url")},
		})
	}

	queryParams := parsedURL.Query()

	switch parsedURL.Host {
	// youtube hosts
	case supportedPlaylistsHost[0], supportedPlaylistsHost[1], supportedPlaylistsHost[2]:

		if parsedURL.Path != "/playlist" || len(queryParams) < 1 || queryParams["list"] == nil {
			log.Println("Invalid YouTubeMusic playlist - " + parsedURL.Path)

			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Errors: Errors{getBadRequestError("Invalid YouTubeMusic playlist url")},
			})
		}

		playlistExists, checkErr := youtube.IsPlaylistValid(queryParams["list"][0])

		if checkErr != nil {
			log.Panicln("youtube.IsPlaylistValid error", checkErr)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if playlistExists {
			log.Println("playlist exists - " + queryParams["list"][0])

			return c.Status(fiber.StatusOK).JSON(&APIResponse{
				Data: map[string]interface{}{
					"isPlaylistValid": true,
				},
			})
		} else {
			log.Println("playlist not found - " + checkErr.Error())

			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Errors: Errors{getBadRequestError("YouTubeMusic playlist does not exist, it might have been deleted")},
			})
		}
		// spotify host
	case supportedPlaylistsHost[3]:
		pathParts := strings.Split(parsedURL.Path, "/")

		if len(pathParts) < 3 || pathParts[1] != "playlist" {

			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Errors: Errors{getBadRequestError("Invalid Spotify playlist url")},
			})
		}

		playlistExists, checkErr := spotify.IsPlaylistValid(pathParts[2])

		if checkErr != nil {
			log.Panic(checkErr)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if playlistExists {
			return c.Status(fiber.StatusOK).JSON(&APIResponse{
				Data: map[string]interface{}{
					"isPlaylistValid": true,
				},
			})
		} else {

			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Errors: Errors{getBadRequestError("Spotify playlist does not exist, it might have been deleted")},
			})
		}
	}

	// if for any reason we reach here, return a server error
	return c.SendStatus(fiber.StatusInternalServerError)
}

func getBadRequestError(detail string) *ErrorObject {
	return &ErrorObject{
		Status: fiber.StatusBadRequest,
		Title:  "Bad Request",
		Detail: detail,
		Source: &ErrorSource{Parameter: "?url"},
	}
}
