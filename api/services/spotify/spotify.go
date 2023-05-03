package spotify

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/to-dy/music-playlist-converter/api/stores/tokenstore"
)

var spotifyBaseURL = "https://api.spotify.com/v1"

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
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

func GetAccessToken() {
	cli := fiber.Client{}

	// set form request body
	args := fiber.AcquireArgs()
	args.Set("grant_type", "client_credentials")
	args.Set("client_id", os.Getenv("SPOTIFY_CLIENT_ID"))
	args.Set("client_secret", os.Getenv("SPOTIFY_CLIENT_SECRET"))

	// make form request nad use Debug to log request and response details
	res := cli.Post("https://accounts.spotify.com/api/token").Form(args).Debug()

	// get response in string
	var bodyData tokenResponse
	_, _, errs := res.Struct(&bodyData)

	if errs != nil {
		log.Panic(errs)
	}

	tokenstore.GlobalTokenStore.SetToken("spotify", bodyData.AccessToken)

	tt, _ := tokenstore.GlobalTokenStore.GetToken("spotify")

	fmt.Println("Spotify Token from store:", tt)

}

func GetPlaylistTracks(id string) {
	cli := fiber.Client{}

	_, tokenValid := tokenstore.GlobalTokenStore.GetToken("spotify")

	if !tokenValid {
		GetAccessToken()
	}

	token, _ := tokenstore.GlobalTokenStore.GetToken("spotify")

	res := cli.Get(spotifyBaseURL+"/playlists/"+id+"/tracks?fields=total,limit,next,offset,previous,items(track(name,is_local,duration_ms,album(album_type,name),artists(name)))&limit=3").
		Set("Authorization", "Bearer "+token).Debug()

	var bodyData PlaylistTracksResponse
	_, _, errs := res.Struct(&bodyData)

	if errs != nil {
		log.Panic(errs)
	}

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
