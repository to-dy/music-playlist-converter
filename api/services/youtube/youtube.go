package youtube

import (
	"context"
	"log"
	"os"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"

	"github.com/to-dy/music-playlist-converter/api/stores/tokenstore"
)

var once sync.Once

var OauthConfig = &oauth2.Config{
	ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
	ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
	RedirectURL:  os.Getenv("YOUTUBE_REDIRECT_URI"),
	Endpoint:     google.Endpoint,
	Scopes:       []string{youtube.YoutubeForceSslScope},
}

var youtubeService *youtube.Service

func init() {
	once.Do(func() {
		ctx := context.Background()
		service, err := youtube.NewService(ctx, option.WithCredentialsFile("google_credentials.json"), option.WithAPIKey(os.Getenv("YOUTUBE_API_KEY")))
		youtubeService = service

		if err != nil {
			log.Panicf("failed to create YouTube service: %v", err)
		}
	})
}

func StoreOauthToken(token *oauth2.Token) {
	tokenstore.GlobalTokenStore.SetToken(string(tokenstore.YOUTUBE_AC), tokenstore.TokenEntry{
		Token:      token,
		Expiration: token.Expiry,
	})
}

func getOauthToken() (*oauth2.Token, error) {
	token, tokenValid := tokenstore.GlobalTokenStore.GetToken(string(tokenstore.YOUTUBE_AC))

	if tokenValid {
		return token, nil

	} else {
		ts := OauthConfig.TokenSource(context.Background(), token)
		token, err := ts.Token()

		tokenstore.GlobalTokenStore.SetToken(string(tokenstore.YOUTUBE_AC), tokenstore.TokenEntry{
			Token: token,
		})

		return token, err
	}
}

func IsPlaylistValid(id string) bool {
	// youtubeService.Playlists.List([]string{"snippet"}).Id(id).MaxResults(1).Do()
	res, err := youtubeService.Playlists.List([]string{"id"}).Id(id).MaxResults(1).Do()

	if err != nil {
		log.Println(err)
		return false
	}

	return res.Items != nil && len(res.Items) > 0
}

func GetPlaylistTracks(id string) []*youtube.PlaylistItem {
	res, err := youtubeService.PlaylistItems.List([]string{"snippet"}).PlaylistId(id).MaxResults(50).Do()

	if err != nil {
		log.Println(err)

		return nil
	}

	// if there are more tracks fetch them, currently limited 100 tracks
	if len(res.Items) < int(res.PageInfo.TotalResults) {
		res2, err2 := youtubeService.PlaylistItems.List([]string{"snippet"}).PlaylistId(id).MaxResults(50).PageToken(res.NextPageToken).Do()

		if err2 != nil {
			log.Println(err2)

			return nil
		}

		res.Items = append(res.Items, res2.Items...)
	}

	return res.Items
}

func SearchTrack(query string, artist string) (track *youtube.SearchResult, found bool) {
	// search for track on youtube by provided query(artist + track)
	res, err := youtubeService.Search.List([]string{"snippet"}).Q(query).MaxResults(1).Type("video").Do()

	if err != nil {
		log.Println(err)

		return nil, false
	}

	if res.Items != nil && len(res.Items) > 0 {
		return res.Items[0], true

	}

	return nil, false
}

func CreatePlaylist(name string) (id *string) {
	token, tokenErr := getOauthToken()

	if tokenErr != nil {
		log.Println(tokenErr)
		return nil
	}

	// ctx := context.Background()

	// service, svErr := youtube.NewService(ctx, option.WithTokenSource(OauthConfig.TokenSource(ctx, token)))

	// if svErr != nil {
	// 	log.Println(svErr)
	// 	return nil
	// }

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
		return nil
	}

	return &res.Id
}

// func AddTracksToPlaylist(playlistId string, tracks []*youtube.SearchResult) *string {
// }
