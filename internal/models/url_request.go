package models

import "time"

// CreateURLRequest represents the request body for creating a short URL
type CreateURLRequest struct {
	URL       string     `json:"url" binding:"required,url"`        // Gin validation: required and must be valid URL
	ExpiresAt *time.Time `json:"expires_at,omitempty"`               // Optional expiration date
	ShortCode *string    `json:"short_code,omitempty"`                // Optional custom short code
}

