package grantTypes

type GrantType string

const (
	AuthorizationCode GrantType = "authorization_code"
	ClientCredentials GrantType = "client_credentials"
)
