package token

// RefreshRequest is used by token endpoints to refresh tokens.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
