package session

import (
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
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
