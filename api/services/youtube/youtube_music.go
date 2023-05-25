package youtube

// youtube music specific api interactions

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/to-dy/music-playlist-converter/api/services/shared_types"
)

type Music struct {
	YoutubeId string
	Title     string
	Artists   shared_types.Artists
	Album     string
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
		},
		"params": "EgWKAQIIAWoKEAoQCRADEAQQBQ%3D%3D", // do not know what this does but it generates the type of data needed
	}

	if len(data) > 0 {
		for _, item := range data {
			body[item.Key] = item.Value
		}
	}

	return body
}

func YTMusic_SearchTrack(query string, artist string) ([]*Music, bool) {
	// search for track on youtube by provided query(artist + track)
	cli := fiber.Client{}

	body := generateBodyContext([]bodyData{{Key: "query", Value: query + " - " + artist}})

	res := cli.Post(YTMusic_BaseURL+"/search?alt=json&maxResults=1&key="+YOUTUBE_MUSIC_KEY).
		UserAgent("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)").
		Set("origin", "https://music.youtube.com").
		JSON(body)

	var ytmRes YTMusic_SearchResults

	_, _, errs := res.Struct(&ytmRes)

	if errs != nil {
		log.Panic(errs)
		// return
	}

	music := parseSearchMusicsBody(&ytmRes)
	musicBytes, _ := json.Marshal(music)

	// for _, item := range music {
	// 	di := &item

	// 	log.Println(di.YoutubeId)
	// }

	filePath := "res_output.json"
	err := os.WriteFile(filePath, musicBytes, 0644)

	if err != nil {
		log.Panic("Error writing file:", err)
	}

	log.Println("JSON data written to", filePath)

	return music, len(music) > 0

}

// func YTMusic_GetPlaylistTracks(id string) ([]*Music, bool) {
func YTMusic_GetPlaylistTracks(id string) {
	cli := fiber.Client{}

	body := generateBodyContext([]bodyData{{Key: "browseId", Value: id}})

	res := cli.Post(YTMusic_BaseURL+"/browse?alt=json&key="+YOUTUBE_MUSIC_KEY).
		UserAgent("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)").
		Set("origin", "https://music.youtube.com").
		JSON(body).Debug()

	_, b, errs := res.Bytes()

	if errs != nil {
		log.Panic(errs)
		// return
	}

	filePath := "res_output_3.json"
	err := os.WriteFile(filePath, b, 0644)

	if err != nil {
		log.Println("Error writing file:", err)
		return
	}

	log.Println("JSON data written to", filePath)

	// music := parseSearchMusicsBody(b)

	// return music, len(music) > 0
}
