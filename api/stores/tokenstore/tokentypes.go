package tokenstore

type TokenName string

const (
	SPOTIFY_CC TokenName = "spotify_client_token"
	SPOTIFY_AC TokenName = "spotify_authorization_code_token"

	YOUTUBE_CC TokenName = "youtube_client_token"
	YOUTUBE_AC TokenName = "youtube_authorization_code_token"
)
