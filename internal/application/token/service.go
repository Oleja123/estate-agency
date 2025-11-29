package token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

type Service interface {
	GenerateAccessToken(userID int, role string, ttl time.Duration) (string, time.Time, error)

	GenerateRefreshToken(userID int, ttl time.Duration) (string, error)

	ValidateAccessToken(token string) (int, string, error)

	ValidateRefreshToken(token string) (int, error)

	InvalidateRefreshToken(token string) error
}

var ErrInvalidToken = errors.New("invalid token")

func genRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

type refreshStore struct {
	mu    sync.RWMutex
	store map[string]refreshEntry
}

type refreshEntry struct {
	UserID int
	Expiry time.Time
}

func newRefreshStore() *refreshStore {
	return &refreshStore{store: make(map[string]refreshEntry)}
}

func (rs *refreshStore) put(token string, entry refreshEntry) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.store[token] = entry
}

func (rs *refreshStore) get(token string) (refreshEntry, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	e, ok := rs.store[token]
	return e, ok
}

func (rs *refreshStore) del(token string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	delete(rs.store, token)
}
