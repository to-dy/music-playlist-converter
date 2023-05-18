package tokenstore

import (
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

var GlobalTokenStore *TokenStore
var once sync.Once

type TokenStore struct {
	store map[string]TokenEntry
	mutex sync.Mutex
}

type TokenEntry struct {
	Token      *oauth2.Token
	Expiration time.Time
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
				Token: &oauth2.Token{
					AccessToken: os.Getenv("YOUTUBE_API_KEY"),
					Expiry:      time.Time{},
				},
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

func (ts *TokenStore) GetToken(name string) (*oauth2.Token, bool) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	entry, found := ts.store[name]
	if !found || entry.Expiration.Before(time.Now()) {
		return nil, false
	}
	return entry.Token, true
}

func (ts *TokenStore) IsTokenValid(name string) bool {
	_, found := ts.GetToken(name)
	return found
}
