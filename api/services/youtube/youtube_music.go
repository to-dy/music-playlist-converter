package youtube

// youtube music specific api interactions

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/to-dy/music-playlist-converter/api/services/shared_types"
)

type Music struct {
	YoutubeId string
	Title     string
	Artists   shared_types.Artists
	Album     shared_types.Album
	Duration  time.Duration
}

type Artist shared_types.Artist

const (
	PageTypeArtist   = "MUSIC_PAGE_TYPE_ARTIST'"
	PageTypeAlbum    = "MUSIC_PAGE_TYPE_ALBUM"
	PageTypePlaylist = "MUSIC_PAGE_TYPE_PLAYLIST"

	// key used on music.youtube.com to access youtube api
	YOUTUBE_MUSIC_KEY = "AIzaSyC9XL3ZjWddXya6X74dJoCTL-WEYFDNX30"
	YTMusic_BaseURL   = "https://music.youtube.com/youtubei/v1"
)

type bodyData struct {
	Key   string
	Value interface{}
}

func generateBodyContext(data []bodyData) map[string]interface{} {
	// body := make(map[string]interface{})

	body := map[string]interface{}{
		"context": map[string]interface{}{
			"capabilities": map[string]interface{}{},
			"client": map[string]string{
				"clientName":    "WEB_REMIX",
				"clientVersion": "0.1",
			},
		}}

	if len(data) > 0 {
		for _, item := range data {
			body[item.Key] = item.Value
		}
	}

	return body
}

func YTMusic_SearchTrack(query string, artist string) (*Music, bool, error) {
	// search for track on youtube by provided query(artist + track)
	cli := fiber.Client{}

	body := generateBodyContext([]bodyData{
		{Key: "query", Value: query + " - " + artist},
		{Key: "params", Value: "EgWKAQIIAWoKEAoQCRADEAQQBQ%3D%3D"}, // do not know what this does, but it generates the type of data needed
	})

	res := cli.Post(YTMusic_BaseURL+"/search?alt=json&maxResults=1&key="+YOUTUBE_MUSIC_KEY).
		UserAgent("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)").
		Set("origin", "https://music.youtube.com").
		JSON(body)

	var ytmRes YTMusic_SearchResults

	_, _, errs := res.Struct(&ytmRes)

	if errs != nil {
		log.Panic(errs)
		return nil, false, errs[0]
	}

	music := parseSearchMusicsBody(&ytmRes)

	// musicBytes, _ := json.Marshal(music)

	// filePath := "res_output.json"
	// err := os.WriteFile(filePath, musicBytes, 0644)

	// if err != nil {
	// 	log.Panic("Error writing file:", err)
	// }

	// log.Println("JSON data written to", filePath)

	if len(music) > 0 {

		return music[0], true, nil
	}

	return nil, false, errors.New("track not found")
}

func YTMusic_GetPlaylistTracks(id string) ([]*Music, error) {
	var tracks = []*Music{}

	cli := fiber.Client{}

	if !strings.HasPrefix(id, "VL") {
		id = "VL" + id
	}

	allowedNumberOfConversions, intConvErr := strconv.Atoi(os.Getenv("ALLOWED_NUMBER_OF_CONVERSIONS"))

	if intConvErr != nil {
		log.Println(intConvErr)
		return nil, intConvErr
	}

	body := generateBodyContext([]bodyData{{Key: "browseId", Value: id}})

	res := cli.Post(YTMusic_BaseURL+"/browse?alt=json&key="+YOUTUBE_MUSIC_KEY).
		UserAgent("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)").
		Set("origin", "https://music.youtube.com").
		JSON(body).Debug()

	var ytmRes YTMusic_PlaylistResults

	status, _, errs := res.Struct(&ytmRes)

	if errs != nil {
		log.Panic(errs)

		return nil, errs[0]
	}

	// TODO: support fetching more tracks via continuations
	//  continuations := ytmRes.Contents.SingleColumnBrowseResultsRenderer.Tabs[0].TabRenderer.Content.SectionListRenderer.Continuations

	if status == http.StatusOK {

		tracks = parseListMusicsFromPlaylistBody(&ytmRes)

		// allowedNumberOfConversions = 0 means convert all tracks
		if allowedNumberOfConversions != 0 {
			// check so we don't go out of range
			if allowedNumberOfConversions > len(tracks) {
				allowedNumberOfConversions = len(tracks)
			}

			tracks = tracks[0:allowedNumberOfConversions]
		}

		return tracks, nil
	}

	return nil, errors.New("playlist not found | Status code: " + strconv.Itoa(status))
}
