package entities

import "time"

// URL represents a shortened URL entity in the database
type URL struct {
	ID          string     `json:"id"` // UUID
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	UserID      *string    `json:"user_id,omitempty"` // Pointer allows nil (for anonymous URLs), UUID
	ClickCount  int        `json:"click_count"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"` // Pointer allows nil (no expiration)
}

