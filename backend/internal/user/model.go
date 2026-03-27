package user

import "github.com/google/uuid"

type User struct {
	ID                uuid.UUID `json:"id"`
	Email             string    `json:"email"`
	PreferredLanguage string    `json:"preferredLanguage"`
}

type RegisterRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	PreferredLanguage string `json:"preferredLanguage,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
