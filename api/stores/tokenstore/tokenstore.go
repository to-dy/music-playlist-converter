package tokenstore

import (
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
	Token      string
	Expiration time.Time
}

func init() {
	once.Do(func() {
		GlobalTokenStore = NewTokenStore()
	})
}

func NewTokenStore() *TokenStore {
	return &TokenStore{
		// store: make(map[string]TokenEntry),

		store: map[string]TokenEntry{
			"spotify": {
				Token:      "",
				Expiration: time.Now(),
			},

			"youtube": {
				Token:      "",
				Expiration: time.Now(),
			},
		},
	}
}

func (ts *TokenStore) SetToken(name, token string) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	ts.store[name] = TokenEntry{
		Token:      token,
		Expiration: time.Now().Add(time.Hour),
	}
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
