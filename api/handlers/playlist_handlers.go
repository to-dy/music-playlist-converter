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

type supportedPlaylist struct {
	YouTube string
	Spotify string
}

var supportedPlaylistsHost = []string{"music.youtube.com", "www.youtube.com", "youtube.com", "open.spotify.com"}

var supportedConversions = &supportedPlaylist{
	YouTube: "youtube",
	Spotify: "spotify",
}

var supportedConversionsList = []string{supportedConversions.YouTube, supportedConversions.Spotify}

type sessionPlaylist struct {
	Id         string
	Title      string
	Url        string
	Source     string
	TrackCount int
	NewTitle   string
}

func VerifyPlaylist(c *fiber.Ctx) error {
	urlQ := c.Query("url")

	if urlQ == "" {
		log.Println("url query parameter is empty")

		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
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

		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
			Errors: Errors{getBadRequestError("Invalid or unsupported playlist url", nil)},
		})
	}

	queryParams := parsedURL.Query()

	switch parsedURL.Host {
	// youtube hosts
	case supportedPlaylistsHost[0], supportedPlaylistsHost[1], supportedPlaylistsHost[2]:

		if parsedURL.Path != "/playlist" || len(queryParams) < 1 || queryParams["list"] == nil {
			log.Println("Invalid YouTubeMusic playlist - " + parsedURL.Path)

			return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
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
				Source:     supportedConversions.YouTube,
				TrackCount: int(playlist.ContentDetails.ItemCount),
			}

			if err := startSession(c, pl); err != nil {
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			return c.Status(fiber.StatusOK).JSON(&ApiOkResponse{
				Data: map[string]interface{}{
					"isPlaylistValid":      true,
					"supportedConversions": []string{supportedConversions.Spotify},
				},
			})
		} else {
			log.Println("playlist not found - ", checkErr)

			return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
				Errors: Errors{getBadRequestError("YouTubeMusic playlist does not exist, it might have been deleted", nil)},
			})
		}
		// spotify host
	case supportedPlaylistsHost[3]:
		pathParts := strings.Split(parsedURL.Path, "/")

		if len(pathParts) < 3 || pathParts[1] != "playlist" {

			return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
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
				Source:     supportedConversions.Spotify,
				TrackCount: int(playlist.Tracks.Total),
			}

			if err := startSession(c, pl); err != nil {
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			return c.Status(fiber.StatusOK).JSON(&ApiOkResponse{
				Data: map[string]interface{}{
					"isPlaylistValid":      true,
					"supportedConversions": []string{supportedConversions.YouTube},
				},
			})
		} else {

			return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
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
	token := sess.Get(session.AuthCodeToken)
	// value set on oauth start
	convertTo := sess.Get(session.ConvertTo)

	if sessionUrl == nil || token == nil || playlistSource != nil || playlistName == nil || playlistTracksCount == nil {
		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
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

	return c.Status(fiber.StatusOK).JSON(&ApiOkResponse{Data: bodyRes})
}

/*
converts valid playlist url to a supported source

it uses the PlaylistURL stored in the session
*/
func ConvertPlaylist(c *fiber.Ctx) error {
	sess, err := session.Store.Get(c)
	if err != nil {
		log.Println("Error getting session - " + err.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	bodyData := struct {
		Title string `json:"title"`
	}{}

	c.Accepts(fiber.MIMEApplicationJSON)

	if err := c.BodyParser(&bodyData); err != nil {
		log.Println("Error parsing body - " + err.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	if bodyData.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
			Errors: Errors{getBadRequestError("title is required", &ErrorSource{Parameter: "title"})},
		})
	}

	sessionUrl := sess.Get(session.PlaylistURL)
	token := sess.Get(session.AuthCodeToken)
	convertTo := sess.Get(session.ConvertTo)
	playlistSource := sess.Get(session.PlaylistSource)
	playlistName := sess.Get(session.PlaylistName)
	playlistTracksCount := sess.Get(session.PlaylistTracksCount)
	playlistId := sess.Get(session.PlaylistID)

	if sessionUrl == nil || token == nil || convertTo == nil || playlistSource == nil || playlistName == nil || playlistTracksCount == nil {
		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
			Errors: Errors{getBadRequestError("invalid session", &ErrorSource{})},
		})
	}

	if !arrutil.Contains(supportedConversionsList, convertTo) {
		log.Println("unsupported playlist conversion - " + convertTo.(string))

		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
			Errors: Errors{getBadRequestError("unsupported playlist conversion source", &ErrorSource{})},
		})
	}

	playlistInfo := &sessionPlaylist{
		Id:         playlistId.(string),
		Title:      playlistName.(string),
		Url:        sessionUrl.(string),
		Source:     playlistSource.(string),
		TrackCount: playlistTracksCount.(int),
		NewTitle:   bodyData.Title,
	}

	// platform we want to convert to
	switch convertTo {
	// youtube
	case supportedConversions.YouTube:
		// youtube -> spotify
		if playlistInfo.Source == supportedConversions.Spotify {
			return youtubeToSpotify(c, playlistInfo, sess.ID())
		}

		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
			Errors: Errors{getBadRequestError("playlist conversion from Youtube to "+playlistInfo.Source+" not supported", &ErrorSource{})},
		})

		// spotify
	case supportedConversions.Spotify:
		//  spotify -> youtube
		if playlistInfo.Source == supportedConversions.YouTube {
			return spotifyToYoutube(c, playlistInfo, sess.ID())
		}

		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
			Errors: Errors{getBadRequestError("playlist conversion from Youtube to "+playlistInfo.Source+" not supported", &ErrorSource{})},
		})

	}

	// if for any reason we reach here, return a server error
	return c.SendStatus(fiber.StatusInternalServerError)
}

func youtubeToSpotify(c *fiber.Ctx, playlistInfo *sessionPlaylist, sessionId string) error {
	spotifyUserId, userIdErr := spotify.GetUserId(sessionId)
	if userIdErr != nil {
		log.Println("spotifyUserId error", userIdErr)

		return c.SendStatus(fiber.StatusInternalServerError)
	}

	ytTracks, getTracksErr := youtube.YTMusic_GetPlaylistTracks(playlistInfo.Id)

	if getTracksErr != nil {
		log.Println("youtubeToSpotify YTMusic_GetPlaylistTracks error", getTracksErr)

		return c.SendStatus(fiber.StatusInternalServerError)
	}

	tracks := youtube.ToSearchTrackList(ytTracks)

	spotifyPlId, cr8plErr := spotify.CreatePlaylist(playlistInfo.NewTitle, spotifyUserId, sessionId)

	if cr8plErr != nil {
		log.Println("spotify.CreatePlaylist error", cr8plErr)

		return c.SendStatus(fiber.StatusInternalServerError)
	}

	addErr := spotify.AddTracksToPlaylist(spotifyPlId, tracks, sessionId)

	if addErr != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusCreated).JSON(&ApiOkResponse{Data: map[string]interface{}{
		"playlistUrl": "https://open.spotify.com/playlist/" + spotifyPlId,
	}})
}

func spotifyToYoutube(c *fiber.Ctx, playlistInfo *sessionPlaylist, sessionId string) error {
	spTracks, getTracksErr := spotify.GetPlaylistTracks(playlistInfo.Id)
	if getTracksErr != nil {
		log.Println("spotifyToYoutube GetPlaylistTracks error", getTracksErr)

		return c.SendStatus(fiber.StatusInternalServerError)
	}

	tracks := spotify.ToSearchTrackList(spTracks)

	ytPlId, createErr := youtube.CreatePlaylist(playlistInfo.NewTitle, sessionId)

	if createErr != nil {
		log.Println("youtube.CreatePlaylist error", createErr)

		return c.SendStatus(fiber.StatusInternalServerError)
	}

	addErr := youtube.AddTracksToPlaylist(ytPlId, tracks, sessionId)

	if addErr != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusCreated).JSON(&ApiOkResponse{Data: map[string]interface{}{
		"playlistUrl": "https://music.youtube.com/playlist?list=" + ytPlId,
	}})
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
