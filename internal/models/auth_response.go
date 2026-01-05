package models

import "time"

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	UserID    string    `json:"user_id"` // UUID
	Email     string    `json:"email"`
	Name      *string   `json:"name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Token     string    `json:"token"` // JWT token
}

// RegisterResponse represents the response after user registration
type RegisterResponse struct {
	Message string      `json:"message"`
	User    AuthResponse `json:"user"`
}

