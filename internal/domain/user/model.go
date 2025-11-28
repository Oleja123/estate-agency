package user

import (
	"fmt"
	"strings"
	"time"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleClient Role = "client"
)

// ParseRole converts a string to a Role, validating allowed values.
func ParseRole(s string) (Role, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "admin":
		return RoleAdmin, nil
	case "client":
		return RoleClient, nil
	default:
		return Role(""), fmt.Errorf("unknown role: %s", s)
	}
}

type User struct {
	Id           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	PhoneNumber  *string   `json:"phone_number"`
	UserRole     Role      `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
