package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gookit/goutil/arrutil"

	"github.com/to-dy/music-playlist-converter/api/services"
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
	NewSource  string
	SessionId  string
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

			return c.Status(fiber.StatusNotFound).JSON(ApiErrorResponse{
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

			return c.Status(fiber.StatusNotFound).JSON(ApiErrorResponse{
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
	c.Accepts(fiber.MIMEApplicationJSON)

	bodyData := struct {
		Title string `json:"title"`
	}{}

	if err := c.BodyParser(&bodyData); err != nil {
		log.Println("Error parsing body - " + err.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	if bodyData.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
			Errors: Errors{getBadRequestError("title is required", &ErrorSource{Parameter: "title"})},
		})
	}

	playlistInfo, handlePlInfoErr := getSessionPlaylistInfo(c, bodyData.Title)
	if handlePlInfoErr != nil {
		handlePlInfoErr()
	}

	return handleConvertPlaylist(c, playlistInfo, nil)
}

/*
SSE handler
converts valid playlist url to a supported source while streaming the process to the client

it uses the PlaylistURL stored in the session
*/
func StreamConvertPlaylist(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-cache")

	// validation checks

	qTitle := c.Query("title")
	if qTitle == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
			Errors: Errors{getBadRequestError("title is required", &ErrorSource{Parameter: "?title"})},
		})
	}

	playlistInfo, handlePlInfoErr := getSessionPlaylistInfo(c, qTitle)
	if handlePlInfoErr != nil {
		handlePlInfoErr()
	}

	// validations passed, get ready to start steaming

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderConnection, "keep-alive")

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		// w.Flush()
		// go handleConvertPlaylist(c, playlistInfo, w)
		handleConvertPlaylist(c, playlistInfo, w)
	})

	return nil
}

func streamEvent(w *bufio.Writer, event string, data string) error {
	ev := fmt.Sprintf("event: %s\n", event)
	dt := fmt.Sprintf("data: %s\n", data)

	w.Write([]byte(ev))
	w.Write([]byte(dt))

	if err := w.Flush(); err != nil {
		log.Println("Error flushing writer - " + err.Error())
		return err
	}

	return nil
}

func handleConvertPlaylist(c *fiber.Ctx, playlistInfo *sessionPlaylist, streamWriter *bufio.Writer) error {
	// platform we want to convert to
	switch playlistInfo.NewSource {
	// youtube
	case supportedConversions.YouTube:
		// youtube -> spotify
		if playlistInfo.Source == supportedConversions.Spotify {
			return youtubeToSpotify(c, playlistInfo, streamWriter)
		}

		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
			Errors: Errors{getBadRequestError("playlist conversion from Youtube to "+playlistInfo.Source+" not supported", &ErrorSource{})},
		})

		// spotify
	case supportedConversions.Spotify:
		//  spotify -> youtube
		if playlistInfo.Source == supportedConversions.YouTube {
			return spotifyToYoutube(c, playlistInfo, streamWriter)
		}

		return c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
			Errors: Errors{getBadRequestError("playlist conversion from Youtube to "+playlistInfo.Source+" not supported", &ErrorSource{})},
		})
	}

	// if for any reason we reach here, return a server error
	return c.SendStatus(fiber.StatusInternalServerError)
}

func getSessionPlaylistInfo(c *fiber.Ctx, newTitle string) (*sessionPlaylist, func()) {
	sess, err := session.Store.Get(c)
	if err != nil {
		log.Println("Error getting session - " + err.Error())
		return nil, func() {
			c.SendStatus(fiber.StatusInternalServerError)
		}
		// return nil, err
	}

	sessionUrl := sess.Get(session.PlaylistURL)
	token := sess.Get(session.AuthCodeToken)
	convertTo := sess.Get(session.ConvertTo)
	playlistSource := sess.Get(session.PlaylistSource)
	playlistName := sess.Get(session.PlaylistName)
	playlistTracksCount := sess.Get(session.PlaylistTracksCount)
	playlistId := sess.Get(session.PlaylistID)

	if sessionUrl == nil || token == nil || convertTo == nil || playlistSource == nil || playlistName == nil || playlistTracksCount == nil {

		return nil, func() {
			c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
				Errors: Errors{getBadRequestError("invalid session", &ErrorSource{})},
			})
		}
	}

	if !arrutil.Contains(supportedConversionsList, convertTo) {
		log.Println("unsupported playlist conversion - " + convertTo.(string))

		return nil, func() {
			c.Status(fiber.StatusBadRequest).JSON(ApiErrorResponse{
				Errors: Errors{getBadRequestError("unsupported playlist conversion source", &ErrorSource{})},
			})
		}

	}

	playlistInfo := &sessionPlaylist{
		Id:         playlistId.(string),
		Title:      playlistName.(string),
		Url:        sessionUrl.(string),
		Source:     playlistSource.(string),
		TrackCount: playlistTracksCount.(int),
		NewTitle:   newTitle,
		NewSource:  convertTo.(string),
		SessionId:  sess.ID(),
	}

	return playlistInfo, nil
}

func youtubeToSpotify(c *fiber.Ctx, playlistInfo *sessionPlaylist, streamWriter *bufio.Writer) error {
	spotifyUserId, userIdErr := spotify.GetUserId(playlistInfo.SessionId)
	if userIdErr != nil {
		log.Println("spotifyUserId error", userIdErr)

		if streamWriter != nil {
			streamEvent(streamWriter, "error", "error getting users spotify user id ")

			d, _ := json.Marshal(fiber.Map{
				"message": "Conversion process aborted",
				"info": fiber.Map{
					"tracks_found":          0,
					"tracks_not_found":      0,
					"conversion_successful": false,
				},
			})

			// tell client to close connection
			streamEvent(streamWriter, "done", string(d))
		} else {

			return c.SendStatus(fiber.StatusInternalServerError)
		}
	}

	ytTracks, getTracksErr := youtube.YTMusic_GetPlaylistTracks(playlistInfo.Id)

	if getTracksErr != nil {
		log.Println("youtubeToSpotify YTMusic_GetPlaylistTracks error", getTracksErr)

		if streamWriter != nil {
			streamEvent(streamWriter, "error", "error getting playlist on YouTubeMusic")

			d, _ := json.Marshal(fiber.Map{
				"message": "Conversion process aborted",
				"info": fiber.Map{
					"tracks_found":          0,
					"tracks_not_found":      0,
					"conversion_successful": false,
				},
			})
			// tell client to close connection
			streamEvent(streamWriter, "done", string(d))
		} else {

			return c.SendStatus(fiber.StatusInternalServerError)
		}
	}

	tracks := youtube.ToSearchTrackList(ytTracks)

	if streamWriter != nil {
		streamEvent(streamWriter, "info", "creating playlist on Spotify")
	}

	spotifyPlId, cr8plErr := spotify.CreatePlaylist(playlistInfo.NewTitle, spotifyUserId, playlistInfo.SessionId)

	if cr8plErr != nil {
		log.Println("spotify.CreatePlaylist error", cr8plErr)

		if streamWriter != nil {
			streamEvent(streamWriter, "error", "error creating playlist on Spotify")

			d, _ := json.Marshal(fiber.Map{
				"message": "Conversion process aborted",
				"info": fiber.Map{
					"tracks_found":          0,
					"tracks_not_found":      0,
					"conversion_successful": false,
				},
			})
			// tell client to close connection
			streamEvent(streamWriter, "done", string(d))
		} else {

			return c.SendStatus(fiber.StatusInternalServerError)
		}
	}

	if streamWriter != nil {
		streamEvent(streamWriter, "info", "Playlist created Spotify")

		// go youtubeToSpotifyStream(tracks, playlistInfo, spotifyPlId, streamWriter)
		youtubeToSpotifyStream(tracks, playlistInfo, spotifyPlId, streamWriter)
	} else {

		addErr := spotify.AddTracksToPlaylist(spotifyPlId, tracks, playlistInfo.SessionId)

		if addErr != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.Status(fiber.StatusCreated).JSON(&ApiOkResponse{Data: map[string]interface{}{
			"playlistUrl": "https://open.spotify.com/playlist/" + spotifyPlId,
		}})
	}

	// if for any reason we reach here, return a server error
	return c.SendStatus(fiber.StatusInternalServerError)
}

func youtubeToSpotifyStream(tracks []*services.SearchTrack, playlistInfo *sessionPlaylist, spotifyPlId string, streamWriter *bufio.Writer) error {
	streamEvent(streamWriter, "info", "Preparing to add tracks to playlist on Spotify")

	uris := []string{}

	for _, track := range tracks {
		trackJson, mErr := json.Marshal(track)
		if mErr != nil {
			log.Println("error marshaling track: ", mErr)

			streamEvent(streamWriter, "track_search", "error encountered with track: "+string(trackJson))
			continue
		}

		artist := ""
		if len(track.Artists) > 0 {
			artist = track.Artists[0].Name
		}

		d, _ := json.Marshal(fiber.Map{
			"message": "Searching track on Spotify: " + track.Title + " by " + artist,
			"track":   track,
			"status":  "searching",
		})
		streamEvent(streamWriter, "track_search", string(d))

		entry, found, err := spotify.SearchTrack(track.Title, artist)

		if err != nil {
			log.Println("error searching track: ", err)

			d, _ := json.Marshal(fiber.Map{
				"message": "error searching for track",
				"track":   track,
				"success": false,
				"status":  "error",
			})
			streamEvent(streamWriter, "track_search", string(d))

			continue
		}

		if !found {
			d, _ := json.Marshal(fiber.Map{
				"message": "track not found",
				"track":   track,
				"success": false,
				"status":  "error",
			})
			streamEvent(streamWriter, "track_search", string(d))

			continue
		}

		if entry != nil {
			uris = append(uris, entry.Uri)

			d, _ := json.Marshal(fiber.Map{
				"message": "Track found",
				"track":   track,
				"success": true,
				"status":  "done",
			})
			streamEvent(streamWriter, "track_search", string(d))
		}
	}

	addTracksErr := spotify.AddTracksToPlaylistByUris(spotifyPlId, uris, playlistInfo.SessionId)

	if addTracksErr != nil {
		streamEvent(streamWriter, "error", "error adding tracks to playlist")
	}

	d, _ := json.Marshal(fiber.Map{
		"message": "Conversion process complete",
		"info": fiber.Map{
			"tracks_found":          len(uris),
			"tracks_not_found":      len(tracks) - len(uris),
			"conversion_successful": len(uris) == len(tracks) && addTracksErr == nil,
		},
	})

	// tell client to close connection
	streamEvent(streamWriter, "done", string(d))

	return nil
}

func spotifyToYoutube(c *fiber.Ctx, playlistInfo *sessionPlaylist, streamWriter *bufio.Writer) error {
	spTracks, getTracksErr := spotify.GetPlaylistTracks(playlistInfo.Id)
	if getTracksErr != nil {
		log.Println("spotifyToYoutube GetPlaylistTracks error", getTracksErr)

		if streamWriter != nil {
			streamEvent(streamWriter, "error", "Unable to get playlist tracks")

			d, _ := json.Marshal(fiber.Map{
				"message": "Conversion process aborted",
				"info": fiber.Map{
					"tracks_found":          0,
					"tracks_not_found":      0,
					"conversion_successful": false,
				},
			})

			// tell client to close connection
			streamEvent(streamWriter, "done", string(d))
		} else {

			return c.SendStatus(fiber.StatusInternalServerError)
		}
	}

	tracks := spotify.ToSearchTrackList(spTracks)

	if streamWriter != nil {
		streamEvent(streamWriter, "info", "Creating playlist on YoutubeMusic")
	}

	ytPlId, createErr := youtube.CreatePlaylist(playlistInfo.NewTitle, playlistInfo.SessionId)

	if createErr != nil {
		if streamWriter != nil {
			log.Println("youtube.CreatePlaylist error", createErr)

			streamEvent(streamWriter, "error", "Error creating playlist on YoutubeMusic")

			d, _ := json.Marshal(fiber.Map{
				"message": "Conversion process aborted",
				"info": fiber.Map{
					"tracks_found":          0,
					"tracks_not_found":      0,
					"conversion_successful": false,
				},
			})
			// tell client to close connection
			streamEvent(streamWriter, "done", string(d))
		} else {

			return c.SendStatus(fiber.StatusInternalServerError)
		}
	}
	if streamWriter != nil {
		streamEvent(streamWriter, "info", "Playlist created Spotify")

		spotifyToYoutubeStream(tracks, playlistInfo, ytPlId, streamWriter)
	} else {

		addErr := youtube.AddTracksToPlaylist(ytPlId, tracks, playlistInfo.SessionId)

		if addErr != nil {

			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.Status(fiber.StatusCreated).JSON(&ApiOkResponse{Data: map[string]interface{}{
			"playlistUrl": "https://music.youtube.com/playlist?list=" + ytPlId,
		}})
	}

	// if for any reason we reach here, return a server error
	return c.SendStatus(fiber.StatusInternalServerError)
}

func spotifyToYoutubeStream(tracks []*services.SearchTrack, playlistInfo *sessionPlaylist, youtubePlId string, streamWriter *bufio.Writer) error {
	streamEvent(streamWriter, "info", "Preparing to add tracks to playlist on YoutubeMusic")

	tracksFound := 0
	tracksNotFound := 0
	var addErr error

	for _, track := range tracks {
		trackJson, mErr := json.Marshal(track)
		if mErr != nil {
			log.Println("error marshaling track: ", mErr)

			streamEvent(streamWriter, "track_search", "error encountered with track: "+string(trackJson))
			continue
		}

		artist := ""
		if len(track.Artists) > 0 {
			artist = track.Artists[0].Name
		}

		d, _ := json.Marshal(fiber.Map{
			"message": "Searching for track on YoutubeMusic: " + track.Title + " by " + artist,
			"track":   track,
			"status":  "searching",
		})
		streamEvent(streamWriter, "track_search", string(d))
		entry, found, sErr := youtube.YTMusic_SearchTrack(track.Title, artist)

		if sErr != nil {
			log.Println("error searching track: ", sErr)

			d, _ := json.Marshal(fiber.Map{
				"message": "error searching for track",
				"track":   track,
				"success": false,
				"status":  "error",
			})
			streamEvent(streamWriter, "track_search", string(d))

			continue
		}

		if !found {
			tracksNotFound++

			d, _ := json.Marshal(fiber.Map{
				"message": "track not found",
				"track":   track,
				"success": false,
				"status":  "error",
			})
			streamEvent(streamWriter, "track_search", string(d))

			continue
		}

		dt, _ := json.Marshal(fiber.Map{
			"message": "Track found",
			"track":   track,
			"success": true,
			"status":  "done",
		})

		streamEvent(streamWriter, "track_search", string(dt))
		streamEvent(streamWriter, "info", "Adding track to playlist on YoutubeMusic")

		addErr = youtube.AddTrackToPlaylist(youtubePlId, entry, playlistInfo.SessionId)

		if addErr != nil {
			streamEvent(streamWriter, "error", "error adding track to playlist")
		}

		tracksFound++
		streamEvent(streamWriter, "info", "track added to playlist on YoutubeMusic")
	}

	d, _ := json.Marshal(fiber.Map{
		"message": "Conversion process complete",
		"info": fiber.Map{
			"tracks_found":          tracksFound,
			"tracks_not_found":      tracksNotFound,
			"conversion_successful": tracksFound == len(tracks) && addErr == nil,
		},
	})

	// tell client to close connection
	streamEvent(streamWriter, "done", string(d))

	return nil
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
