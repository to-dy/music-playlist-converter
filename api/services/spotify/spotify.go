package spotify

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	spotifyOauth "golang.org/x/oauth2/spotify"

	"github.com/to-dy/music-playlist-converter/api/services"
	"github.com/to-dy/music-playlist-converter/api/services/shared_types"
	"github.com/to-dy/music-playlist-converter/api/stores/tokenstore"
)

var spotifyBaseURL = "https://api.spotify.com/v1"

var OauthConfig = &oauth2.Config{
	ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
	ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
	RedirectURL:  os.Getenv("SPOTIFY_REDIRECT_URI"),
	Endpoint:     spotifyOauth.Endpoint,
	Scopes:       []string{"playlist-modify-public", "playlist-read-private"},
}

var clientCredentialsConfig = &clientcredentials.Config{
	ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
	ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
	TokenURL:     spotifyOauth.Endpoint.TokenURL,
	AuthStyle:    oauth2.AuthStyleInParams,
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`

	// comes only with authorization code flow
	Scope        *string `json:"scope"`
	RefreshToken *string `json:"refresh_token"`
}

type Track struct {
	Album    shared_types.Album `json:"album"`
	Artists  shared_types.Artists
	Duration int    `json:"duration_ms"`
	IsLocal  bool   `json:"is_local"`
	Name     string `json:"name"`
	Uri      string `json:"uri"`
}

type Artist shared_types.Artist

type Item struct {
	Track Track `json:"track"`
}

type PlaylistTracksResponse struct {
	Items    []Item `json:"items"`
	Limit    int    `json:"limit"`
	Next     string `json:"next"`
	Offset   int    `json:"offset"`
	Previous string `json:"previous"`
	Total    int    `json:"total"`
}

func StoreOauthToken(token *oauth2.Token) {
	tokenstore.GlobalTokenStore.SetToken(string(tokenstore.SPOTIFY_AC), tokenstore.TokenEntry{
		Token:      token,
		Expiration: token.Expiry,
	})

}

func fetchCredentialToken() (string, error) {
	// cli := fiber.Client{}
	// args := fiber.AcquireArgs()

	// // set form request body
	// args.Set("grant_type", string(grantTypes.ClientCredentials))
	// args.Set("client_id", os.Getenv("SPOTIFY_CLIENT_ID"))
	// args.Set("client_secret", os.Getenv("SPOTIFY_CLIENT_SECRET"))

	// // make form request and use Debug to log request and response details
	// res := cli.Post("https://accounts.spotify.com/api/token").Form(args).Debug()

	// // get response in string
	// var bodyData tokenResponse
	// _, _, errs := res.Struct(&bodyData)

	// if errs != nil {
	// 	log.Panic(errs)

	// 	return "", errs[0]
	// }

	// tokenstore.GlobalTokenStore.SetToken(string(tokenstore.SPOTIFY_CC), tokenstore.TokenEntry{
	// 	Token:      bodyData.AccessToken,
	// 	Expiration: time.Now().Add(time.Second * time.Duration(bodyData.ExpiresIn)),
	// })

	token, err := clientCredentialsConfig.Token(context.Background())

	tokenstore.GlobalTokenStore.SetToken(string(tokenstore.SPOTIFY_CC), tokenstore.TokenEntry{
		Token:      token,
		Expiration: token.Expiry,
	})

	return token.AccessToken, err
}

func getAccessToken(name string) (string, error) {
	token, tokenValid := tokenstore.GlobalTokenStore.GetToken(name)

	if tokenValid {
		return token.AccessToken, nil
	}

	if !tokenValid && name == string(tokenstore.SPOTIFY_CC) {
		token, err := fetchCredentialToken()

		return token, err

	}

	if !tokenValid && name == string(tokenstore.SPOTIFY_AC) {
		ts := OauthConfig.TokenSource(context.Background(), token)
		token, err := ts.Token()

		tokenstore.GlobalTokenStore.SetToken(string(tokenstore.SPOTIFY_AC), tokenstore.TokenEntry{
			Token:      token,
			Expiration: token.Expiry,
		})

		return token.AccessToken, err

	}

	return "", errors.New("unknown token name")
}

// verify if spotify playlist exists
func IsPlaylistValid(id string) bool {
	cli := fiber.Client{}

	token, _ := getAccessToken(string(tokenstore.SPOTIFY_CC))

	res := cli.Get(spotifyBaseURL+"/playlists/"+id+"?fields=id").
		Set("Authorization", "Bearer "+token).Debug()

	var bodyData struct {
		Id *string `json:"id"`
	}
	status, _, errs := res.Struct(&bodyData)

	if errs != nil {
		log.Panic(errs)
		return false
	}

	if status == http.StatusOK {
		return bodyData.Id != nil
	}

	return false
}

func GetPlaylistTracks(id string) []Item {
	cli := fiber.Client{}

	token, _ := getAccessToken(string(tokenstore.SPOTIFY_CC))

	res := cli.Get(spotifyBaseURL+"/playlists/"+id+"/tracks?limit=50&fields=total,limit,next,offset,previous,items(track(name,is_local,duration_ms,album(album_type,name),artists(name)))").
		Set("Authorization", "Bearer "+token).Debug()

	var bodyData PlaylistTracksResponse
	_, _, errs := res.Struct(&bodyData)

	if errs != nil {
		log.Panic(errs)
	}

	// tracks := make([]Item, 0, bodyData.Total)
	tracks := bodyData.Items
	// tracks := arrutil.Map(bodyData.Items, func(item Item) (track Track, find bool) {
	// 	return item.Track, true
	// })

	// if there are more tracks fetch them, currently limited 100 tracks
	if len(tracks) < bodyData.Total {
		_, _, errs = cli.Get(bodyData.Next).Set("Authorization", "Bearer "+token).Debug().
			Struct(&bodyData)

		if errs != nil {
			log.Panic(errs)
		}

		tracks = append(tracks, bodyData.Items...)

	}

	return tracks
}

func ToSearchTrackList(tracks []*Item) *services.SearchTrackList {
	searchTrackList := make(services.SearchTrackList, 0, len(tracks))

	for _, track := range tracks {
		t := services.SearchTrack{
			Title:    track.Track.Name,
			Artists:  track.Track.Artists,
			Album:    track.Track.Album,
			Duration: int64(track.Track.Duration),
		}

		searchTrackList = append(searchTrackList, t)
	}

	return &searchTrackList
}

func SearchTrack(query string, artist string) (track *Track, found bool) {
	cli := fiber.Client{}

	token, _ := getAccessToken(string(tokenstore.SPOTIFY_CC))

	makeQuery := url.QueryEscape("track:" + query + " artist:" + artist)

	res := cli.Get(spotifyBaseURL+"/search?q="+makeQuery+"&type=track&limit=1").
		Set("Authorization", "Bearer "+token).Debug()

	var bodyData PlaylistTracksResponse

	status, _, errs := res.Struct(&bodyData)
	if errs != nil {
		log.Panic(errs)
	}

	if status == http.StatusOK && len(bodyData.Items) > 0 {
		return &bodyData.Items[0].Track, true
	}

	return nil, false
}

func CreatePlaylist(name string, userId string) (id *string, err []error) {
	cli := fiber.Client{}

	token, _ := getAccessToken(string(tokenstore.SPOTIFY_AC))

	body := map[string]string{
		"name": name,
	}

	res := cli.Post(spotifyBaseURL+"/users/"+userId+"/playlists").
		Set("Authorization", "Bearer "+token).
		JSON(body).Debug()

	var bodyData struct {
		Id  string `json:"id"`
		URI string `json:"uri"`
	}

	status, _, errs := res.Struct(&bodyData)

	if errs != nil {
		log.Panic(errs)
		return id, err
	}

	if status == http.StatusCreated {
		return &bodyData.Id, nil
	}

	return id, errs
}

func AddTracksToPlaylist(playlistId string, tracks services.SearchTrackList) *string {
	cli := fiber.Client{}

	token, _ := getAccessToken(string(tokenstore.SPOTIFY_AC))

	uris := ""

	for _, track := range tracks {
		artist := ""
		if len(track.Artists) > 0 {
			artist = track.Artists[0].Name
		}

		entry, found := SearchTrack(track.Title, artist)

		if found {
			uris += entry.Uri + ","
		}
	}

	body := map[string]string{
		"uris": uris,
	}

	res := cli.Post(spotifyBaseURL+"/playlists/"+playlistId+"/tracks").Set("Authorization", "Bearer "+token).
		JSON(body).Debug()

	var bodyData struct {
		SnapshotID string `json:"snapshot_id"`
	}

	status, _, errs := res.Struct(&bodyData)

	if errs != nil {
		log.Panic(errs)
		return nil
	}

	if status == http.StatusCreated {
		return &bodyData.SnapshotID
	}

	return nil
}

// handle unauthorized response
func handleUnauthorized(agent *fiber.Agent, tokenName tokenstore.TokenName) *fiber.Agent {
	// refetch accessToken and try again
	token, _ := getAccessToken(string(tokenName))

	return agent.Reuse().Set("Authorization", "Bearer "+token)
}
