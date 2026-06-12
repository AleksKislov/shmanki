package user

import "github.com/google/uuid"

type User struct {
	ID                uuid.UUID `json:"id"`
	Email             string    `json:"email"`
	DisplayName       string    `json:"displayName"`
	PreferredLanguage string    `json:"preferredLanguage"`
	IsAdmin           bool      `json:"isAdmin,omitempty"`
}

type RegisterRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	PreferredLanguage string `json:"preferredLanguage,omitempty"`
}

type UpdatePreferredLanguageRequest struct {
	PreferredLanguage string `json:"preferredLanguage"`
}

type UpdateProfileRequest struct {
	PreferredLanguage string `json:"preferredLanguage,omitempty"`
	DisplayName       string `json:"displayName,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
