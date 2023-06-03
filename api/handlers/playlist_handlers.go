package handlers

import (
	"log"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gookit/goutil/arrutil"

	"github.com/to-dy/music-playlist-converter/api/services/spotify"
	"github.com/to-dy/music-playlist-converter/api/services/youtube"
	"github.com/to-dy/music-playlist-converter/api/stores/session"
)

var supportedPlaylistsHost = []string{"music.youtube.com", "www.youtube.com", "youtube.com", "open.spotify.com"}

var supportedConversions = []string{"youtube", "spotify"}

type sessionPlaylist struct {
	Id         string
	Title      string
	Url        string
	Source     string
	TrackCount int
}

func VerifyPlaylist(c *fiber.Ctx) error {
	urlQ := c.Query("url")

	if urlQ == "" {
		log.Println("url query parameter is empty")

		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Errors: Errors{getBadRequestError("The `url` query parameter is required.", nil)},
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
			Errors: Errors{getBadRequestError("Invalid or unsupported playlist url", nil)},
		})
	}

	queryParams := parsedURL.Query()

	switch parsedURL.Host {
	// youtube hosts
	case supportedPlaylistsHost[0], supportedPlaylistsHost[1], supportedPlaylistsHost[2]:

		if parsedURL.Path != "/playlist" || len(queryParams) < 1 || queryParams["list"] == nil {
			log.Println("Invalid YouTubeMusic playlist - " + parsedURL.Path)

			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Errors: Errors{getBadRequestError("Invalid YouTubeMusic playlist url", nil)},
			})
		}

		playlist, checkErr := youtube.FindPlaylist(queryParams["list"][0])

		if checkErr != nil {
			log.Println("youtube.FindPlaylist error", checkErr)
			// TODO: check and handle based on error type
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if playlist != nil {
			log.Println("playlist exists - " + queryParams["list"][0])

			pl := &sessionPlaylist{
				Id:         playlist.Id,
				Title:      playlist.Snippet.Title,
				Url:        parsedURL.String(),
				Source:     supportedConversions[0],
				TrackCount: int(playlist.ContentDetails.ItemCount),
			}

			if err := startSession(c, pl); err != nil {
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			return c.Status(fiber.StatusOK).JSON(&APIResponse{
				Data: map[string]interface{}{
					"isPlaylistValid": true,
				},
			})
		} else {
			log.Println("playlist not found - ", checkErr)

			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Errors: Errors{getBadRequestError("YouTubeMusic playlist does not exist, it might have been deleted", nil)},
			})
		}
		// spotify host
	case supportedPlaylistsHost[3]:
		pathParts := strings.Split(parsedURL.Path, "/")

		if len(pathParts) < 3 || pathParts[1] != "playlist" {

			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Errors: Errors{getBadRequestError("Invalid Spotify playlist url", nil)},
			})
		}

		playlist, checkErr := spotify.FindPlaylist(pathParts[2])

		if checkErr != nil {
			log.Println("spotify.FindPlaylist error", checkErr)
			// TODO: check and handle based on error type
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if playlist != nil {
			pl := &sessionPlaylist{
				Id:         playlist.Id,
				Title:      playlist.Name,
				Url:        parsedURL.String(),
				Source:     supportedConversions[1],
				TrackCount: int(playlist.Tracks.Total),
			}

			if err := startSession(c, pl); err != nil {
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			return c.Status(fiber.StatusOK).JSON(&APIResponse{
				Data: map[string]interface{}{
					"isPlaylistValid": true,
				},
			})
		} else {

			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Errors: Errors{getBadRequestError("Spotify playlist does not exist, it might have been deleted", nil)},
			})
		}
	}

	// if for any reason we reach here, return a server error
	return c.SendStatus(fiber.StatusInternalServerError)
}

func PreviewPlaylistConversion(c *fiber.Ctx) error {
	// check  user session
	sess, err := session.Store.Get(c)
	if err != nil {
		log.Println("Error getting session - " + err.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	// check for the playlist details and auth token in session
	sessionUrl := sess.Get(session.PlaylistURL)
	playlistSource := sess.Get(session.PlaylistSource)
	playlistName := sess.Get(session.PlaylistName)
	playlistTracksCount := sess.Get(session.PlaylistTracksCount)
	// value set on oauth start
	convertTo := sess.Get(session.ConvertTo)

	var token string

	if convertTo != nil && convertTo == supportedPlaylistsHost[1] /* youtube */ {
		token = sess.Get(session.SpotifyAuthCodeToken).(string)
	} else if convertTo != nil && convertTo == supportedPlaylistsHost[0] /* spotify */ {
		token = sess.Get(session.YoutubeAuthCodeToken).(string)
	}

	if sessionUrl == nil || token == "" || playlistSource != nil || playlistName == nil || playlistTracksCount == nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Errors: Errors{getBadRequestError("invalid session", &ErrorSource{})},
		})
	}

	bodyRes := map[string]interface{}{
		"playlistUrl":         sessionUrl,
		"playlistName":        playlistName,
		"playlistTracksCount": playlistTracksCount,
		"source":              playlistSource,
		"convertTo":           convertTo,
	}

	return c.Status(fiber.StatusOK).JSON(&APIResponse{Data: bodyRes})
}

/*
		/convert?to=youtube, /convert?to=spotify
	 converts valid playlist url to a supported source

	 it uses the url stored in the session from VerifyPlaylist()
*/
func ConvertPlaylist(c *fiber.Ctx) error {
	to := c.Query("to")

	if to == "" {
		log.Println("to query parameter is empty")

		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Errors: Errors{getBadRequestError("The `to` query parameter is required.", &ErrorSource{Parameter: "?to"})},
		})
	}

	if !arrutil.Contains(supportedConversions, to) {
		log.Println("unsupported playlist conversion - " + to)

		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Errors: Errors{getBadRequestError("Invalid or unsupported playlist conversion", &ErrorSource{Parameter: "?to"})},
		})
	}

	sess, err := session.Store.Get(c)
	if err != nil {
		log.Println("Error getting session - " + err.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	sessionUrl := sess.Get(session.PlaylistURL)

	if sessionUrl == nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Errors: Errors{getBadRequestError("invalid session", &ErrorSource{Parameter: "session"})},
		})
	}

	switch to {
	case supportedConversions[0]:

		return nil
	case supportedConversions[1]:
		return nil

	}

	// if for any reason we reach here, return a server error
	return c.SendStatus(fiber.StatusInternalServerError)
}

func startSession(c *fiber.Ctx, pl *sessionPlaylist) error {
	sess, err := session.Store.Get(c)

	if err != nil {
		log.Println("Error getting session - " + err.Error())
		return err
	}

	sess.Set(session.PlaylistID, pl.Id)
	sess.Set(session.PlaylistURL, pl.Url)
	sess.Set(session.PlaylistSource, pl.Source)
	sess.Set(session.PlaylistName, pl.Title)
	sess.Set(session.PlaylistTracksCount, pl.TrackCount)

	return sess.Save()
}

func getBadRequestError(detail string, source *ErrorSource) *ErrorObject {
	var src *ErrorSource

	if source != nil {
		src = source
	} else {
		src = &ErrorSource{Parameter: "?url"}
	}

	return &ErrorObject{
		Status: fiber.StatusBadRequest,
		Title:  "Bad Request",
		Detail: detail,
		Source: src,
	}
}
