package youtube

// youtube music specific api interactions

import (
	"errors"
	"log"
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
	Album     string
	Duration  time.Duration
}

type Artist shared_types.Artist

const (
	PageTypeArtist   = "MUSIC_PAGE_TYPE_ARTIST'"
	PageTypeAlbum    = "MUSIC_PAGE_TYPE_ALBUM"
	PageTypePlaylist = "MUSIC_PAGE_TYPE_PLAYLIST"
	YTMusic_BaseURL  = "https://music.youtube.com/youtubei/v1"
)

// reference https://github.com/baptisteArno/node-youtube-music/blob/main/src/searchMusics.ts#L6 *modified
func parseSearchMusicsBody(body interface{}) []*Music {
	results := make([]*Music, 0)

	// Extract the necessary data from the body JSON object contents.tabbedSearchResultsRenderer.tabs[0].tabRenderer.content.sectionListRenderer.contents
	if contents, ok := body.(map[string]interface{})["contents"].(map[string]interface{})["tabbedSearchResultsRenderer"].(map[string]interface{})["tabs"].([]interface{})[0].(map[string]interface{})["tabRenderer"].(map[string]interface{})["content"].(map[string]interface{})["sectionListRenderer"].(map[string]interface{})["contents"].([]interface{}); ok {
		// contents[0].musicShelfRenderer | extract musicShelfRenderer from the first object in contents in array
		musicShelfRenderer := contents[0].(map[string]interface{})["musicShelfRenderer"].(map[string]interface{})

		// Iterate over `contents` in musicShelfRenderer and parse each music item
		for _, content := range musicShelfRenderer["contents"].([]interface{}) {
			contentMap := content.(map[string]interface{})
			song, err := parseMusicItem(contentMap)
			if err != nil {
				log.Println("Error parsing music item:", err)
				continue
			}
			if song != nil {
				results = append(results, song)
			}
		}
	}

	return results
}

// reference: https://github.com/baptisteArno/node-youtube-music/blob/main/src/parsers.ts#L59
func parseMusicItem(content map[string]interface{}) (*Music, error) {
	var youtubeId, title, album string
	var artists shared_types.Artists

	var duration *time.Duration

	// content.musicResponsiveListItemRenderer | Get the music responsive list item renderer.
	musicResponsiveListItemRenderer := content["musicResponsiveListItemRenderer"].(map[string]interface{})

	// content.musicResponsiveListItemRenderer.flexColumns | Get the flex columns.
	flexColumns, okk := musicResponsiveListItemRenderer["flexColumns"].([]interface{})

	if len(flexColumns) > 0 {
		if firstFlexColumn, ok := flexColumns[0].(map[string]interface{})["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{}); ok {
			if runs, ok := firstFlexColumn["text"].(map[string]interface{})["runs"].([]interface{}); ok && len(runs) > 0 {
				// Extract title
				title = runs[0].(map[string]interface{})["text"].(string)

				// Extract YouTube ID
				if navigationEndpoint, ok := runs[0].(map[string]interface{})["navigationEndpoint"].(map[string]interface{}); ok {
					if watchEndpoint, ok := navigationEndpoint["watchEndpoint"].(map[string]interface{}); ok {
						youtubeId = watchEndpoint["videoId"].(string)
					}
				}
			}
		}

		// content.musicResponsiveListItemRenderer.flexColumns[1]
		if len(flexColumns) > 1 {
			if secondFlexColumn, ok := flexColumns[1].(map[string]interface{})["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{}); ok {

				if runs, ok := secondFlexColumn["text"].(map[string]interface{})["runs"].([]interface{}); ok {
					// Extract artists
					artists = listArtists(runs)

					// Extract duration
					if length := len(runs); length > 0 {
						label := runs[length-1].(map[string]interface{})["text"].(string)
						duration = parseDuration(label)
					}

					// Extract album
					if length := len(runs); length > 2 {
						album = runs[length-3].(map[string]interface{})["text"].(string)
					}
				}

			}
		}
	}

	if !okk {
		return nil, errors.New("unable to parse music item")
	}

	return &Music{
		YoutubeId: youtubeId,
		Title:     title,
		Artists:   artists,
		Album:     album,
		Duration:  *duration,
	}, nil
}

func parseDuration(durationLabel string) *time.Duration {
	durationList := strings.Split(durationLabel, ":")

	var hours, minutes, seconds int

	if len(durationList) == 3 {
		hours = getInt(durationList[0])
		minutes = getInt(durationList[1])
		seconds = getInt(durationList[2])
	} else if len(durationList) == 2 {
		minutes = getInt(durationList[0])
		seconds = getInt(durationList[1])
	} else {
		t := time.Duration(0)

		return &t
	}

	t := time.Duration(hours*3600 + minutes*60 + seconds)
	return &t
}

func getInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// reference: https://github.com/baptisteArno/node-youtube-music/blob/main/src/parsers.ts#L33
func listArtists(data []interface{}) shared_types.Artists {
	// Create a new slice to store the artists.
	artists := make(shared_types.Artists, 0)

	// Iterate over the data.
	for _, item := range data {
		// Check if the item is a navigation endpoint.
		if navigationEndpoint, ok := item.(map[string]interface{})["navigationEndpoint"].(map[string]interface{}); ok {
			// Check if the navigation endpoint has a browse endpoint.
			if browseEndpoint, ok := navigationEndpoint["browseEndpoint"].(map[string]interface{}); ok {
				// Check if the browse endpoint has a browse endpoint context supported configs.
				if browseEndpointContextSupportedConfigs, ok := browseEndpoint["browseEndpointContextSupportedConfigs"].(map[string]interface{}); ok {
					// Check if the browse endpoint context supported configs has a browse endpoint context music config.
					if browseEndpointContextMusicConfig, ok := browseEndpointContextSupportedConfigs["browseEndpointContextMusicConfig"].(map[string]interface{}); ok {
						// Check if the browse endpoint context music config has a page type.
						if pageType, ok := browseEndpointContextMusicConfig["pageType"].(string); ok {
							// Check if the page type is equal to the artist page type.
							if pageType == PageTypeArtist {
								// Add the artist to the artists slice.
								artists = append(artists, shared_types.Artist{Name: item.(map[string]interface{})["text"].(string)})
							}
						}
					}
				}
			}
		}
	}

	// If the artists slice is empty, check if there is a delimiter in the data.
	if len(artists) == 0 {
		delimiterIndex := -1
		for i, item := range data {
			if item.(map[string]interface{})["text"].(string) == " • " {
				delimiterIndex = i
				break
			}
		}

		if delimiterIndex != -1 {
			for _, item := range data[:delimiterIndex] {
				if item.(map[string]interface{})["name"].(string) != " & " {
					artists = append(artists, shared_types.Artist{
						Name: item.(map[string]interface{})["text"].(string),
					})
				}
			}
		}
	}

	// Return the artists slice.
	return artists
}

func YTMusic_SearchTrack(query string, artist string) ([]*Music, bool) {
	// search for track on youtube by provided query(artist + track)
	cli := fiber.Client{}

	body := map[string]interface{}{
		"context": map[string]interface{}{
			"capabilities": map[string]interface{}{},
			"client": map[string]string{
				"clientName":    "WEB_REMIX",
				"clientVersion": "0.1",
			},
		},
		"params": "EgWKAQIIAWoKEAoQCRADEAQQBQ%3D%3D", // not what this does but it generates the type of data needed
		"query":  query,
	}

	res := cli.Post(YTMusic_BaseURL+"/search?alt=json&maxResults=1&key="+os.Getenv("YOUTUBE_API_KEY")).
		UserAgent("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)").
		Set("origin", "https://music.youtube.com").
		JSON(body).Debug()

	_, b, errs := res.Bytes()

	if errs != nil {
		log.Panic(errs)
		// return
	}

	music := parseSearchMusicsBody(b)

	return music, len(music) > 0

	// filePath := "res_output_2.json"
	// err := os.WriteFile(filePath, b, 0644)

	// if err != nil {
	// 	log.Println("Error writing file:", err)
	// 	return
	// }
	// log.Println("JSON data written to", filePath)
}
