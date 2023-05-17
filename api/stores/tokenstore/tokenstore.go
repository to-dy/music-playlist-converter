package tokenstore

import (
	"os"
	"sync"
	"time"
)

var GlobalTokenStore *TokenStore
var once sync.Once

type TokenStore struct {
	store map[string]TokenEntry
	mutex sync.Mutex
}

type TokenEntry struct {
	Token        string
	Scope        *string `json:"scope"`
	RefreshToken *string `json:"refresh_token"`
	Expiration   time.Time
}

func init() {
	once.Do(func() {
		GlobalTokenStore = NewTokenStore()
	})
}

func NewTokenStore() *TokenStore {
	return &TokenStore{

		store: map[string]TokenEntry{
			string(YOUTUBE_CC): {
				Token:      os.Getenv("YOUTUBE_API_KEY"),
				Expiration: time.Time{}, // indefinite, key doesn't expire unless revoked ,
			},
		},
	}
}

func (ts *TokenStore) SetToken(name string, token TokenEntry) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	ts.store[name] = token
}

func (ts *TokenStore) GetToken(name string) (string, bool) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	entry, found := ts.store[name]
	if !found || entry.Expiration.Before(time.Now()) {
		return "", false
	}
	return entry.Token, true
}

func (ts *TokenStore) IsTokenValid(name string) bool {
	_, found := ts.GetToken(name)
	return found
}
