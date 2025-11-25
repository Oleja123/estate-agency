package token

import (
	"time"
)

type memoryTokenService struct {
	access  *refreshStore // reuse structure for token->entry storage
	refresh *refreshStore
}

// NewMemoryService creates an in-memory token service. Not for production.
func NewMemoryService() Service {
	return &memoryTokenService{access: newRefreshStore(), refresh: newRefreshStore()}
}

func (m *memoryTokenService) GenerateAccessToken(userID int, ttl time.Duration) (string, time.Time, error) {
	t, err := genRandomString(32)
	if err != nil {
		return "", time.Time{}, err
	}
	exp := time.Now().Add(ttl)
	m.access.put(t, refreshEntry{UserID: userID, Expiry: exp})
	return t, exp, nil
}

func (m *memoryTokenService) GenerateRefreshToken(userID int, ttl time.Duration) (string, error) {
	t, err := genRandomString(48)
	if err != nil {
		return "", err
	}
	exp := time.Now().Add(ttl)
	m.refresh.put(t, refreshEntry{UserID: userID, Expiry: exp})
	return t, nil
}

func (m *memoryTokenService) ValidateAccessToken(token string) (int, error) {
	e, ok := m.access.get(token)
	if !ok || time.Now().After(e.Expiry) {
		return 0, ErrInvalidToken
	}
	return e.UserID, nil
}

func (m *memoryTokenService) ValidateRefreshToken(token string) (int, error) {
	e, ok := m.refresh.get(token)
	if !ok || time.Now().After(e.Expiry) {
		return 0, ErrInvalidToken
	}
	return e.UserID, nil
}

func (m *memoryTokenService) InvalidateRefreshToken(token string) error {
	m.refresh.del(token)
	return nil
}
