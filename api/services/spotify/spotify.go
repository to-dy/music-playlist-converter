package spotify

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	spotifyOauth "golang.org/x/oauth2/spotify"

	"github.com/to-dy/music-playlist-converter/api/services"
	"github.com/to-dy/music-playlist-converter/api/services/shared_types"
	"github.com/to-dy/music-playlist-converter/api/stores/tokenstore"
	"github.com/to-dy/music-playlist-converter/initializers"
)

var spotifyBaseURL = "https://api.spotify.com/v1"

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

type PlaylistResponse struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Tracks struct {
		Total int `json:"total"`
	} `json:"tracks"`
}

type PlaylistTracksResponse struct {
	Items    []*Item `json:"items"`
	Limit    int     `json:"limit"`
	Next     string  `json:"next"`
	Offset   int     `json:"offset"`
	Previous string  `json:"previous"`
	Total    int     `json:"total"`
}

var OauthConfig *oauth2.Config
var clientCredentialsConfig *clientcredentials.Config

func init() {
	// TODO: investigate why I have to call LoadEnv to access env vars here
	initializers.LoadEnv()

	OauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("SPOTIFY_REDIRECT_URI"),
		Endpoint:     spotifyOauth.Endpoint,
		Scopes:       []string{"playlist-modify-public", "playlist-read-private"},
	}

	clientCredentialsConfig = &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		TokenURL:     spotifyOauth.Endpoint.TokenURL,
		AuthStyle:    oauth2.AuthStyleInParams,
	}
}

func StoreAuthCodeToken(token *oauth2.Token, sessionId string) {
	prefix := sessionId + "_"

	tokenstore.GlobalTokenStore.SetToken(prefix+string(tokenstore.SPOTIFY_AC), tokenstore.TokenEntry{
		Token:      token,
		Expiration: token.Expiry,
	})

}

func fetchClientToken() (string, error) {
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

func getAuthCodeToken(sessionId string) (string, error) {
	token, tokenValid := tokenstore.GlobalTokenStore.GetToken(sessionId + "_" + string(tokenstore.SPOTIFY_AC))

	if tokenValid {
		return token.AccessToken, nil
	}

	if !tokenValid && token != nil {
		// refresh token
		ts := OauthConfig.TokenSource(context.Background(), token)
		token, err := ts.Token()

		StoreAuthCodeToken(token, sessionId)

		return token.AccessToken, err

	}

	return "", errors.New("unknown token name")
}

func getClientToken() (string, error) {
	token, tokenValid := tokenstore.GlobalTokenStore.GetToken(string(tokenstore.SPOTIFY_CC))

	if tokenValid {
		return token.AccessToken, nil
	} else {
		token, err := fetchClientToken()

		return token, err
	}
}

func GetUserId(sessionId string) (string, error) {
	cli := fiber.Client{}

	token, tokenErr := getAuthCodeToken(sessionId)

	if tokenErr != nil {
		return "", tokenErr
	}

	res := cli.Get(spotifyBaseURL+"/me").
		Set("Authorization", "Bearer "+token).Debug()

	var bodyData struct {
		Id string `json:"id"`
	}

	status, b, errs := res.Struct(&bodyData)

	log.Println("spotify/me : ", string(b))

	if errs != nil {
		log.Panic(errs)
		return "", errs[0]
	}

	if status == http.StatusOK {
		return bodyData.Id, nil
	}

	return "", errors.New("error getting spotify user id | status code: " + fmt.Sprint(status))
}

// verify if spotify playlist exists
func FindPlaylist(id string) (*PlaylistResponse, error) {
	cli := fiber.Client{}

	token, tokenErr := getClientToken()

	if tokenErr != nil {
		return nil, tokenErr
	}

	res := cli.Get(spotifyBaseURL+"/playlists/"+id+"?fields=id,name,tracks(total)").
		Set("Authorization", "Bearer "+token).Debug()

	var bodyData PlaylistResponse

	status, _, errs := res.Struct(&bodyData)

	if errs != nil {
		log.Panic(errs)
		return nil, errs[0]
	}

	if status == http.StatusOK {
		return &bodyData, nil
	}

	return nil, errors.New("error verifying playlist | status code: " + fmt.Sprint(status))
}

func GetPlaylistTracks(id string) ([]*Item, error) {
	cli := fiber.Client{}

	token, _ := getClientToken()

	allowedNumberOfConversions, intConvErr := strconv.Atoi(os.Getenv("ALLOWED_NUMBER_OF_CONVERSIONS"))

	if intConvErr != nil {
		log.Println(intConvErr)
		return nil, intConvErr
	}

	res := cli.Get(spotifyBaseURL+"/playlists/"+id+"/tracks?limit=50&fields=total,limit,next,offset,previous,items(track(name,is_local,duration_ms,album(album_type,name),artists(name)))").
		Set("Authorization", "Bearer "+token).Debug()

	var bodyData PlaylistTracksResponse
	_, _, errs := res.Struct(&bodyData)

	if errs != nil {
		log.Panic(errs)
	}

	tracks := bodyData.Items

	// TODO: update this implementation to properly fetch more tracks, possibly using a while loop

	// if there are more tracks fetch them
	if (len(tracks) < allowedNumberOfConversions) && (len(tracks) < bodyData.Total) {
		_, _, errs = cli.Get(bodyData.Next).Set("Authorization", "Bearer "+token).Debug().
			Struct(&bodyData)

		if errs != nil {
			log.Panic(errs)
		}

		tracks = append(tracks, bodyData.Items...)

	}

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

func ToSearchTrackList(tracks []*Item) services.SearchTrackList {
	searchTrackList := make(services.SearchTrackList, 0, len(tracks))

	for _, track := range tracks {
		t := services.SearchTrack{
			Title:    track.Track.Name,
			Artists:  track.Track.Artists,
			Album:    track.Track.Album,
			Duration: int64(track.Track.Duration),
		}

		searchTrackList = append(searchTrackList, &t)
	}

	return searchTrackList
}

func SearchTrack(query string, artist string) (*Track, bool, error) {
	cli := fiber.Client{}

	token, _ := getClientToken()

	makeQuery := url.QueryEscape("track:" + query + " artist:" + artist)

	res := cli.Get(spotifyBaseURL+"/search?q="+makeQuery+"&type=track&limit=1").
		Set("Authorization", "Bearer "+token).Debug()

	var bodyData PlaylistTracksResponse

	status, _, errs := res.Struct(&bodyData)
	if errs != nil {
		return nil, false, errs[0]
	}

	if status == http.StatusOK {
		if len(bodyData.Items) > 0 {
			return &bodyData.Items[0].Track, true, nil
		}

		return nil, false, errors.New("track not found")
	}

	return nil, false, errors.New("error searching track | status code: " + fmt.Sprint(status))
}

func CreatePlaylist(name string, userId string, sessionId string) (id string, err []error) {
	cli := fiber.Client{}

	token, _ := getAuthCodeToken(sessionId)

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
		return bodyData.Id, nil
	}

	return id, errs
}

func AddTracksToPlaylist(playlistId string, tracks services.SearchTrackList, sessionId string) error {
	uris := []string{}

	for _, track := range tracks {
		artist := ""
		if len(track.Artists) > 0 {
			artist = track.Artists[0].Name
		}

		entry, _, err := SearchTrack(track.Title, artist)
		if err != nil {
			log.Println("error searching track: ", err)

			continue
		}

		if entry != nil {
			uris = append(uris, entry.Uri)
		}
	}

	return AddTracksToPlaylistByUris(playlistId, uris, sessionId)
}

func AddTracksToPlaylistByUris(playlistId string, uris []string, sessionId string) error {
	cli := fiber.Client{}

	token, _ := getAuthCodeToken(sessionId)

	body := map[string]string{
		"uris": strings.Join(uris, ","),
	}

	res := cli.Post(spotifyBaseURL+"/playlists/"+playlistId+"/tracks").Set("Authorization", "Bearer "+token).
		JSON(body).Debug()

	status, _, errs := res.Bytes()

	if errs != nil {
		log.Panic(errs)
		return errs[0]
	}

	if status == http.StatusCreated {
		return nil
	}

	return errors.New("error adding tracks to playlist | status code: " + fmt.Sprint(status))
}
