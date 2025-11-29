package userhandler

type RegisterRequestDoc struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	PhoneNumber string `json:"phone_number"`
}

type LoginRequestDoc struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponseDoc struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type PublicUserDoc struct {
	Id          int     `json:"id"`
	Email       string  `json:"email"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	PhoneNumber *string `json:"phone_number"`
	Role        string  `json:"role"`
	IsActive    bool    `json:"is_active"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type ListUsersResponseDoc struct {
	Users []PublicUserDoc `json:"users"`
	Total int             `json:"total"`
}

type UpdateProfileRequestDoc struct {
	FirstName   *string `json:"first_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	PhoneNumber *string `json:"phone_number,omitempty"`
	Email       *string `json:"email,omitempty"`
	Role        *string `json:"role,omitempty"`
}

type ChangePasswordRequestDoc struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type RoleRequestDoc struct {
	Role string `json:"role"`
}
