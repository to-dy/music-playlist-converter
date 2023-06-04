package youtube

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"

	"github.com/gookit/goutil/arrutil"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"

	"github.com/to-dy/music-playlist-converter/api/services"
	"github.com/to-dy/music-playlist-converter/api/stores/tokenstore"
	"github.com/to-dy/music-playlist-converter/initializers"
)

var once sync.Once

var OauthConfig *oauth2.Config

var youtubeService *youtube.Service

func init() {
	// once.Do(func() {
	// TODO: investigate why I have to call LoadEnv to access env vars here
	initializers.LoadEnv()

	OauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("YOUTUBE_REDIRECT_URI"),
		Endpoint:     google.Endpoint,
		Scopes:       []string{youtube.YoutubeForceSslScope},
	}

	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(os.Getenv("YOUTUBE_API_KEY")))
	youtubeService = service

	if err != nil {
		log.Panicf("failed to create YouTube service: %v", err)
	}
	// })
}

func StoreAuthCodeToken(token *oauth2.Token, sessionId string) {
	prefix := sessionId + "_"

	tokenstore.GlobalTokenStore.SetToken(prefix+string(tokenstore.YOUTUBE_AC), tokenstore.TokenEntry{
		Token:      token,
		Expiration: token.Expiry,
	})
}

func getAuthCodeToken(sessionId string) (*oauth2.Token, error) {
	token, tokenValid := tokenstore.GlobalTokenStore.GetToken(sessionId + "_" + string(tokenstore.YOUTUBE_AC))

	if tokenValid {
		return token, nil
	}

	if !tokenValid && token != nil {
		// refresh token
		ts := OauthConfig.TokenSource(context.Background(), token)
		token, err := ts.Token()

		StoreAuthCodeToken(token, sessionId)

		return token, err
	}

	return nil, errors.New("unknown token name")
}

func FindPlaylist(id string) (*youtube.Playlist, error) {
	res, err := youtubeService.Playlists.List([]string{"id", "contentDetails", "snippet"}).Id(id).MaxResults(1).Do()

	if err != nil {
		log.Println("IsPlaylistValid error" + err.Error())

		return nil, err
	}

	// find playlist by id
	if res.Items != nil && len(res.Items) > 0 {
		found, err := arrutil.Find(res.Items, func(item interface{}) bool {
			pl := item.(*youtube.Playlist)

			return pl.Id == id
		})

		if err != nil {
			log.Panicln("error searching res.Items in IsPlaylistValid")
			return nil, err
		}

		return found.(*youtube.Playlist), nil

	} else {
		return nil, nil
	}
}

func ToSearchTrackList(tracks []*Music) *services.SearchTrackList {
	searchTrackList := make(services.SearchTrackList, 0, len(tracks))

	for _, track := range tracks {
		t := services.SearchTrack{
			Title:    track.Title,
			Artists:  track.Artists,
			Album:    track.Album,
			Duration: track.Duration.Milliseconds(),
		}

		searchTrackList = append(searchTrackList, t)
	}

	return &searchTrackList
}

func CreatePlaylist(name string, sessionId string) (*string, error) {
	token, tokenErr := getAuthCodeToken(sessionId)

	if tokenErr != nil {
		log.Println(tokenErr)
		return nil, tokenErr
	}

	playlist := &youtube.Playlist{
		Snippet: &youtube.PlaylistSnippet{
			Title: name,
		},
	}
	call := youtubeService.Playlists.Insert([]string{"snippet,status"}, playlist)

	call.Header().Add("Authorization", "Bearer "+token.AccessToken)

	res, err := call.Do()

	if err != nil {
		log.Panic(err)

		return nil, err
	}

	return &res.Id, nil
}

func AddTracksToPlaylist(playlistId string, tracks services.SearchTrackList, sessionId string) error {
	token, tokenErr := getAuthCodeToken(sessionId)

	if tokenErr != nil {
		log.Println(tokenErr)
		return tokenErr
	}

	for _, track := range tracks {
		artist := ""
		if len(track.Artists) > 0 {
			artist = track.Artists[0].Name
		}

		entry, found := YTMusic_SearchTrack(track.Title, artist)

		PlaylistItem := &youtube.PlaylistItem{
			Snippet: &youtube.PlaylistItemSnippet{
				ResourceId: &youtube.ResourceId{
					VideoId: entry.YoutubeId,
				},
				Title: entry.Title,
			},
		}

		if found {
			call := youtubeService.PlaylistItems.Insert([]string{"snippet", "status"}, PlaylistItem)

			call.Header().Add("Authorization", "Bearer "+token.AccessToken)

			_, err := call.Do()

			if err != nil {
				log.Println("error adding track to playlist", err)
				continue
			}

		} else {
			continue
		}

	}

	return nil
}
