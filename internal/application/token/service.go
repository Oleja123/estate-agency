package token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

// Service provides token generation and validation.
type Service interface {
	// GenerateAccessToken creates a signed access token and returns token and expiry.
	// The access token will contain the user id and role in its claims.
	GenerateAccessToken(userID int, role string, ttl time.Duration) (string, time.Time, error)
	// GenerateRefreshToken creates a refresh token (opaque) and stores it.
	GenerateRefreshToken(userID int, ttl time.Duration) (string, error)
	// ValidateAccessToken extracts userID and role from access token.
	// Returns userID, role and error.
	ValidateAccessToken(token string) (int, string, error)
	// ValidateRefreshToken checks the refresh token and returns associated userID.
	ValidateRefreshToken(token string) (int, error)
	// InvalidateRefreshToken removes refresh token from store.
	InvalidateRefreshToken(token string) error
}

// ErrInvalidToken is returned when token is invalid.
var ErrInvalidToken = errors.New("invalid token")

// helper to generate secure random strings
func genRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// in-memory refresh store used by implementations
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
