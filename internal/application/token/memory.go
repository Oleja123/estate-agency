package token

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type memoryTokenService struct {
	// secret used to sign access tokens (HMAC). For the in-memory service we
	// generate a random secret at startup. In production you'd use a stable
	// secret or asymmetric keys.
	secret  []byte
	refresh *refreshStore
}

// NewMemoryService creates an in-memory token service. Not for production.
func NewMemoryService() Service {
	// generate random secret
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return &memoryTokenService{secret: b, refresh: newRefreshStore()}
}

func (m *memoryTokenService) GenerateAccessToken(userID int, role string, ttl time.Duration) (string, time.Time, error) {
	exp := time.Now().Add(ttl)
	// Build JWT manually (header.payload.signature) using HMAC-SHA256.
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	payload := map[string]interface{}{
		"sub":  userID,
		"role": role,
		"exp":  exp.Unix(),
		"iat":  time.Now().Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)

	segHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	segPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := segHeader + "." + segPayload

	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(signingInput))
	sig := mac.Sum(nil)
	segSig := base64.RawURLEncoding.EncodeToString(sig)

	tokenStr := signingInput + "." + segSig
	return tokenStr, exp, nil
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

func (m *memoryTokenService) ValidateAccessToken(tokenStr string) (int, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return 0, ErrInvalidToken
	}
	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return 0, ErrInvalidToken
	}
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(signingInput))
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return 0, ErrInvalidToken
	}
	// parse payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, ErrInvalidToken
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return 0, ErrInvalidToken
	}
	// check exp
	expf, ok := payload["exp"].(float64)
	if !ok {
		return 0, ErrInvalidToken
	}
	if time.Now().Unix() > int64(expf) {
		return 0, ErrInvalidToken
	}
	// extract sub
	subf, ok := payload["sub"].(float64)
	if !ok {
		return 0, ErrInvalidToken
	}
	return int(subf), nil
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
