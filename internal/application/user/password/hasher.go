package password

import "golang.org/x/crypto/bcrypt"

// Hasher encapsulates password hashing and comparison.
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type bcryptHasher struct{}

func NewBcryptHasher() Hasher { return &bcryptHasher{} }

func (b *bcryptHasher) Hash(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func (b *bcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
