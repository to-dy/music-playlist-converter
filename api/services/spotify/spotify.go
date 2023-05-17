package spotify

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"

	grantTypes "github.com/to-dy/music-playlist-converter/api/services"
	"github.com/to-dy/music-playlist-converter/api/stores/tokenstore"
	tokenTypes "github.com/to-dy/music-playlist-converter/api/stores/tokenstore"
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
	Album    Album    `json:"album"`
	Artists  []Artist `json:"artists"`
	Duration int      `json:"duration_ms"`
	IsLocal  bool     `json:"is_local"`
	Name     string   `json:"name"`
	Uri      string   `json:"uri"`
}

type Album struct {
	AlbumType string `json:"album_type"`
	Name      string `json:"name"`
}

type Artist struct {
	Name string `json:"name"`
}

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

func FetchAccessToken(name string, code ...string) (string, error) {
	token, tokenValid := tokenstore.GlobalTokenStore.GetToken(name)

	if tokenValid {
		return token, nil

	} else if !tokenValid && len(code) == 0 { // no code provided meaning its a client_credential grant request
		cli := fiber.Client{}
		args := fiber.AcquireArgs()

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

		tokenstore.GlobalTokenStore.SetToken(string(tokenTypes.SPOTIFY_CC), tokenstore.TokenEntry{
			Token:      bodyData.AccessToken,
			Expiration: time.Now().Add(time.Second * time.Duration(bodyData.ExpiresIn)),
		})

		return bodyData.AccessToken, nil

	} else { // authorization code grant request
		cli := fiber.Client{}
		args := fiber.AcquireArgs()

		args.Set("grant_type", string(grantTypes.AuthorizationCode))
		args.Set("code", code[0])
		args.Set("redirect_uri", os.Getenv("SPOTIFY_REDIRECT_URI"))

		res := cli.Post("https://accounts.spotify.com/api/token").Form(args).Debug()

		var bodyData tokenResponse
		_, _, errs := res.Struct(&bodyData)

		if errs != nil {
			log.Panic(errs)
		}

		tokenstore.GlobalTokenStore.SetToken(string(tokenTypes.SPOTIFY_AC), tokenstore.TokenEntry{
			Token:        bodyData.AccessToken,
			Expiration:   time.Now().Add(time.Second * time.Duration(bodyData.ExpiresIn)),
			RefreshToken: bodyData.RefreshToken,
			Scope:        bodyData.Scope,
		})

		return bodyData.AccessToken, nil
	}
}

// verify if spotify playlist exists
func IsPlaylistValid(id string) bool {
	cli := fiber.Client{}

	token, _ := FetchAccessToken(string(tokenTypes.SPOTIFY_CC))

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

	token, _ := FetchAccessToken(string(tokenTypes.SPOTIFY_CC))

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

func SearchTrack(query string, artist string) (track Track, found bool) {
	cli := fiber.Client{}

	token, _ := FetchAccessToken(string(tokenTypes.SPOTIFY_CC))

	makeQuery := url.QueryEscape("track:" + query + " artist:" + artist)

	res := cli.Get(spotifyBaseURL+"/search?q="+makeQuery+"&type=track&limit=1").
		Set("Authorization", "Bearer "+token).Debug()

	var bodyData PlaylistTracksResponse

	status, _, errs := res.Struct(&bodyData)
	if errs != nil {
		log.Panic(errs)
	}

	if status == http.StatusOK {
		return bodyData.Items[0].Track, true
	}

	return Track{}, false
}

func CreatePlaylist(name string, userId string) (id *string) {
	cli := fiber.Client{}

	token, _ := FetchAccessToken(string(tokenTypes.SPOTIFY_AC))

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
		return id
	}

	if status == http.StatusCreated {
		return &bodyData.Id
	}

	return id
}

func AddTracksToPlaylist(playlistId string, tracks []Track) *string {
	cli := fiber.Client{}

	token, _ := FetchAccessToken(string(tokenTypes.SPOTIFY_AC))

	uris := ""

	for _, track := range tracks {
		entry, found := SearchTrack(track.Name, track.Artists[0].Name)

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
		snapshotID string `json:"snapshot_id"`
	}

	status, _, errs := res.Struct(&bodyData)

	if errs != nil {
		log.Panic(errs)
		return nil
	}

	if status == http.StatusCreated {
		return &bodyData.snapshotID
	}

	return nil
}

// handle unauthorized response
func handleUnauthorized(agent *fiber.Agent, tokenName tokenTypes.TokenName) *fiber.Agent {
	// refetch accessToken and try again
	token, _ := FetchAccessToken(string(tokenName))

	return agent.Reuse().Set("Authorization", "Bearer "+token)
}
