package youtube

import (
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/to-dy/music-playlist-converter/api/services/shared_types"
)

// reference https://github.com/baptisteArno/node-youtube-music/blob/main/src/searchMusics.ts#L6 *modified
func parseSearchMusicsBody(body *YTMusic_SearchResults) []*Music {
	results := make([]*Music, 0)

	// Extract the necessary data from the body JSON object contents.tabbedSearchResultsRenderer.tabs[0].tabRenderer.content.sectionListRenderer.contents
	contents := body.Contents.TabbedSearchResultsRenderer.Tabs[0].TabRenderer.Content.SectionListRenderer.Contents

	if len(contents) != 0 {
		// contents[0].musicShelfRenderer | extract musicShelfRenderer from the first object in contents in array
		musicShelfRenderer := contents[0].MusicShelfRenderer

		// Iterate over `contents` in musicShelfRenderer and parse each music item
		for _, content := range musicShelfRenderer.Contents {
			song, err := parseMusicItem(&content)

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

// ref: https://github.com/baptisteArno/node-youtube-music/blob/main/src/listMusicsFromPlaylist.ts#LL6C18-L6C18
func parseListMusicsFromPlaylistBody(body map[string]interface{}) []Music {
	contents := body["contents"].(map[string]interface{})["singleColumnBrowseResultsRenderer"].(map[string]interface{})["tabs"].([]interface{})[0].(map[string]interface{})["tabRenderer"].(map[string]interface{})["content"].(map[string]interface{})["sectionListRenderer"].(map[string]interface{})["contents"].([]interface{})[0].(map[string]interface{})["musicPlaylistShelfRenderer"].(map[string]interface{})["contents"].([]interface{})

	results := []Music{}

	for _, content := range contents {
		contentMap := content.(map[string]interface{})
		song := parseMusicInPlaylistItem(contentMap)
		if song != nil {
			results = append(results, *song)
		}
	}

	return results
}

// reference: https://github.com/baptisteArno/node-youtube-music/blob/main/src/parsers.ts#L59
func parseMusicItem(content *YTMusic_MusicShelfContent) (*Music, error) {
	// func parseMusicItem(content map[string]interface{}) (*Music, error) {
	var youtubeId, title, album string
	var artists shared_types.Artists

	var duration *time.Duration

	// content.musicResponsiveListItemRenderer | Get the music responsive list item renderer.
	musicResponsiveListItemRenderer := content.MusicResponsiveListItemRenderer

	// content.musicResponsiveListItemRenderer.flexColumns | Get the flex columns.
	flexColumns := musicResponsiveListItemRenderer.FlexColumns

	if len(flexColumns) > 0 {
		firstFlexColumn := flexColumns[0].MusicResponsiveListItemFlexColumnRenderer

		if len(firstFlexColumn.Text.Runs) > 0 {
			// Extract title
			title = firstFlexColumn.Text.Runs[0].Text

			// Extract YouTube ID
			_, ok := reflect.TypeOf(firstFlexColumn.Text.Runs[0].NavigationEndpoint).FieldByName("WatchEndpoint")

			if ok {
				youtubeId = firstFlexColumn.Text.Runs[0].NavigationEndpoint.WatchEndpoint.VideoId

			}
		}

		if len(flexColumns) > 1 {
			secondFlexColumn := flexColumns[1].MusicResponsiveListItemFlexColumnRenderer

			runs := secondFlexColumn.Text.Runs

			if len(runs) > 0 {
				// Extract artists
				artists = listArtists(&runs)

				// Extract duration
				if length := len(runs); length > 0 {
					label := runs[length-1].Text
					duration = parseDuration(label)
				}

				// Extract album
				if length := len(runs); length > 2 {
					album = runs[4].Text
				}
			}

		}
	}

	// if !okk {
	// 	return nil, errors.New("unable to parse music item")
	// }

	return &Music{
		YoutubeId: youtubeId,
		Title:     title,
		Artists:   artists,
		Album:     album,
		Duration:  *duration,
	}, nil
}

func parseMusicInPlaylistItem(content map[string]interface{}) *Music {
	musicRenderer := content["musicResponsiveListItemRenderer"].(map[string]interface{})
	flexColumns := musicRenderer["flexColumns"].([]interface{})

	var youtubeId, title string
	var artists []string
	var album string
	var duration *time.Duration

	// Extract YouTube ID
	if len(flexColumns) > 0 {
		flexColumn := flexColumns[0].(map[string]interface{})["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{})
		runs := flexColumn["text"].(map[string]interface{})["runs"].([]interface{})
		if len(runs) > 0 {
			navigationEndpoint := runs[0].(map[string]interface{})["navigationEndpoint"].(map[string]interface{})
			if watchEndpoint, ok := navigationEndpoint["watchEndpoint"].(map[string]interface{}); ok {
				youtubeId = watchEndpoint["videoId"].(string)
			}
		}
	}

	// Extract title
	if len(flexColumns) > 0 {
		flexColumn := flexColumns[0].(map[string]interface{})["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{})
		runs := flexColumn["text"].(map[string]interface{})["runs"].([]interface{})
		if len(runs) > 0 {
			title = runs[0].(map[string]interface{})["text"].(string)
		}
	}

	// Extract artists
	if len(flexColumns) > 1 {
		flexColumn := flexColumns[1].(map[string]interface{})["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{})
		runs := flexColumn["text"].(map[string]interface{})["runs"].([]interface{})
		artists = listArtists(runs)
	}

	// Extract album
	if len(flexColumns) > 2 {
		flexColumn := flexColumns[2].(map[string]interface{})["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{})
		runs := flexColumn["text"].(map[string]interface{})["runs"].([]interface{})
		if len(runs) > 0 {
			album = runs[0].(map[string]interface{})["text"].(string)
		}
	}

	// Extract duration
	if len(flexColumns) > 0 {
		flexColumn := flexColumns[0].(map[string]interface{})["musicResponsiveListItemFixedColumnRenderer"].(map[string]interface{})
		runs := flexColumn["text"].(map[string]interface{})["runs"].([]interface{})
		if len(runs) > 0 {
			label := runs[0].(map[string]interface{})["text"].(string)
			duration = &Duration{
				Label:        label,
				TotalSeconds: parseDuration(label),
			}
		}
	}

	return &Music{
		YoutubeId: youtubeId,
		Title:     title,
		Artists:   artists,
		Album:     album,
		Duration:  duration,
	}
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

// reference: https://github.com/baptisteArno/node-youtube-music/blob/main/src/parsers.ts#L33
func listArtists(data *YTMusic_Runs) shared_types.Artists {
	// Create a new slice to store the artists.
	artists := make(shared_types.Artists, 0)

	// Iterate over the data.
	for _, item := range *data {
		// Check if the navigation endpoint has a browse endpoint.
		_, ok := reflect.TypeOf(item.NavigationEndpoint).FieldByName("BrowseEndpoint")

		if ok {
			// Check if the browse endpoint context supported configs has a browse endpoint context music config.
			browseEndpointContextMusicConfig := item.NavigationEndpoint.BrowseEndpoint.BrowseEndpointContextSupportedConfigs.BrowseEndpointContextMusicConfig

			// Check if the browse endpoint context music config has a page type.
			pageType := browseEndpointContextMusicConfig.PageType

			// Check if the page type is equal to the artist page type.
			if pageType == PageTypeArtist {
				// Add the artist to the artists slice.
				artists = append(artists, shared_types.Artist{Name: item.Text})
			}
		}

	}

	// If the artists slice is empty, check if there is a delimiter in the data.
	if len(artists) == 0 {
		delimiterIndex := -1
		for i, item := range *data {
			if item.Text == " • " {
				delimiterIndex = i
				break
			}
		}

		if delimiterIndex != -1 {
			cdata := *data

			for _, item := range cdata[:delimiterIndex] {
				if item.Text != " & " {
					artists = append(artists, shared_types.Artist{
						Name: item.Text,
					})
				}
			}
		}
	}

	// Return the artists slice.
	return artists
}

func getInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}
