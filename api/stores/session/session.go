package session

import (
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
)

// Session keys
const (
	PlaylistURL         = "pl_url"
	PlaylistSource      = "pl_source"
	PlaylistID          = "pl_id"
	PlaylistName        = "pl_name"
	PlaylistTracksCount = "pl_tracks_count"

	ConvertTo     = "convert_to"
	AuthCodeToken = "spotify_auth_code_token"
)

// var Store *session.Store

// func init() {
// 	// initialize sessions middleware
// 	Store = session.New(session.Config{
// 		Expiration:     time.Hour,
// 		CookieSameSite: "Lax",
// 		CookiePath:     "/",
// 		CookieHTTPOnly: true,
// 	})
// }

var Store *session.Store = session.New(session.Config{
	Expiration:     time.Hour,
	CookieSameSite: "Lax",
	CookiePath:     "/",
	CookieHTTPOnly: true,
})
