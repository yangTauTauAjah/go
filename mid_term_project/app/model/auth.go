package model

// AuthUser adalah identitas yang dibawa access token.
type AuthUser struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email,max=120"`
	Password string `json:"password" validate:"required,min=8"`
}

type TokenResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
	User        AuthUser `json:"user"`
}
