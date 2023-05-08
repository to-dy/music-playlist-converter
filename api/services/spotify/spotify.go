package spotify

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	grantTypes "github.com/to-dy/music-playlist-converter/api/services"
	"github.com/to-dy/music-playlist-converter/api/stores/tokenstore"
)

var spotifyBaseURL = "https://api.spotify.com/v1"

type SpotifyTokenName string

const (
	SPOTIFY_AC SpotifyTokenName = "spotify_ac"
	SPOTIFY_CC SpotifyTokenName = "spotify_cc"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`

	// comes only with authorization code flow
	Scope        *string `json:"scope"`
	RefreshToken *string `json:"refresh_token"`
}

type Track struct {
	Album    Album    `json:"album"`
	Artists  []Artist `json:"artists"`
	Duration int      `json:"duration_ms"`
	IsLocal  bool     `json:"is_local"`
	Name     string   `json:"name"`
}

type Album struct {
	AlbumType string `json:"album_type"`
	Name      string `json:"name"`
}

type Artist struct {
	Name string `json:"name"`
}

type PlaylistTracksResponse struct {
	Items []struct {
		Track Track `json:"track"`
	} `json:"items"`
	Limit    int    `json:"limit"`
	Next     string `json:"next"`
	Offset   int    `json:"offset"`
	Previous string `json:"previous"`
	Total    int    `json:"total"`
}

func FetchAccessToken(code ...string) {
	cli := fiber.Client{}
	args := fiber.AcquireArgs()

	// no code provided meaning its a client_credential grant request
	if len(code) == 0 {
		// set form request body
		args.Set("grant_type", string(grantTypes.ClientCredentials))
		args.Set("client_id", os.Getenv("SPOTIFY_CLIENT_ID"))
		args.Set("client_secret", os.Getenv("SPOTIFY_CLIENT_SECRET"))

		// make form request and use Debug to log request and response details
		res := cli.Post("https://accounts.spotify.com/api/token").Form(args).Debug()

		// get response in string
		var bodyData tokenResponse
		_, _, errs := res.Struct(&bodyData)

		if errs != nil {
			log.Panic(errs)
		}

		tokenstore.GlobalTokenStore.SetToken(string(SPOTIFY_CC), tokenstore.TokenEntry{
			Token:      bodyData.AccessToken,
			Expiration: time.Now().Add(time.Second * time.Duration(bodyData.ExpiresIn)),
		})

		tt, _ := tokenstore.GlobalTokenStore.GetToken(string(SPOTIFY_CC))

		fmt.Println("Spotify Token from store:", tt)
	} else { // authorization code grant request
		args.Set("grant_type", string(grantTypes.AuthorizationCode))
		args.Set("code", code[0])
		args.Set("redirect_uri", os.Getenv("SPOTIFY_REDIRECT_URI"))

		res := cli.Post("https://accounts.spotify.com/api/token").Form(args).Debug()

		var bodyData tokenResponse
		_, _, errs := res.Struct(&bodyData)

		if errs != nil {
			log.Panic(errs)
		}

		tokenstore.GlobalTokenStore.SetToken(string(SPOTIFY_AC), tokenstore.TokenEntry{
			Token:        bodyData.AccessToken,
			Expiration:   time.Now().Add(time.Second * time.Duration(bodyData.ExpiresIn)),
			RefreshToken: bodyData.RefreshToken,
			Scope:        bodyData.Scope,
		})
	}

}

func checkAndObtainToken(name string) string {
	_, tokenValid := tokenstore.GlobalTokenStore.GetToken(name)

	if !tokenValid {
		FetchAccessToken()
	}

	token, _ := tokenstore.GlobalTokenStore.GetToken(name)

	return token

}

func GetPlaylistTracks(id string) {
	cli := fiber.Client{}

	token := checkAndObtainToken(string(SPOTIFY_CC))

	res := cli.Get(spotifyBaseURL+"/playlists/"+id+"/tracks?fields=total,limit,next,offset,previous,items(track(name,is_local,duration_ms,album(album_type,name),artists(name)))&limit=3").
		Set("Authorization", "Bearer "+token).Debug()

	var bodyData PlaylistTracksResponse
	_, _, errs := res.Struct(&bodyData)

	if errs != nil {
		log.Panic(errs)
	}

}

func CreatePlaylistAndAddTracks(name string, userId string) {
	cli := fiber.Client{}

	token := checkAndObtainToken(string(SPOTIFY_AC))

	body := map[string]string{
		"name": name,
	}

	res := cli.Post(spotifyBaseURL+"/users/"+userId+"/playlists").
		Set("Authorization", "Bearer "+token).
		JSON(body).Debug()

	res.String()

}

// testing with "net/http" for learning purposes
func __GetSpotifyAccessToken__() {

	// set form request body
	reqBody := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {string(os.Getenv("SPOTIFY_CLIENT_ID"))},
		"client_secret": {string(os.Getenv("SPOTIFY_CLIENT_SECRET"))},
	}

	// setup request
	req, reqErr := http.NewRequest("POST", spotifyBaseURL+"/token", strings.NewReader(reqBody.Encode()))

	if reqErr != nil {
		// log.Panic(reqErr)
		log.Fatalf("Error creating request: %v", reqErr)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// make request
	res, resErr := http.DefaultClient.Do(req)

	if resErr != nil {
		log.Panic(resErr)
	}
	defer res.Body.Close()

	fmt.Println("Response Status:", res.Status)

	// if res.StatusCode != 200 {
	// 	log.Fatalf("Unexpected status code: %d", res.StatusCode)
	// }

	// print response body in terminal    (for debugging)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response Body:", string(body))
}
