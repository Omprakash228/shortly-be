package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"shortly-be/internal/cache"
	"shortly-be/internal/models"
	"shortly-be/internal/repository"
)

// URLService defines the interface for URL business logic
type URLService interface {
	CreateShortURL(req *models.CreateURLRequest, userID *string, baseURL string) (*models.CreateURLResponse, error)
	GetOriginalURL(shortCode string) (string, error)
	GetURLStats(shortCode string, userID *string) (*models.URLStatsResponse, error)
	GetClickAnalytics(shortCode string, userID *string, hours int) ([]map[string]interface{}, error)
	DeleteURL(shortCode string, userID *string) error
	UpdateExpiresAt(shortCode string, userID *string, expiresAt *time.Time) error
	GetUserURLs(userID string) ([]*models.URLStatsResponse, error)
}

type urlService struct {
	repo  repository.URLRepository
	cache cache.Cache
	ctx   context.Context
}

// NewURLService creates a new URL service
func NewURLService(repo repository.URLRepository, cacheClient cache.Cache) URLService {
	svc := &urlService{
		repo: repo,
		ctx:  context.Background(),
	}
	// Only set cache if provided (allows graceful degradation)
	if cacheClient != nil {
		svc.cache = cacheClient
	}
	return svc
}

// Reserved short codes that cannot be used
var reservedCodes = map[string]bool{
	"admin":    true,
	"api":      true,
	"www":      true,
	"mail":     true,
	"ftp":      true,
	"localhost": true,
	"health":   true,
	"auth":     true,
	"login":    true,
	"register": true,
	"signin":   true,
	"signup":   true,
	"signout":  true,
	"logout":   true,
	"shorten":  true,
	"urls":     true,
	"url":      true,
	"stats":    true,
	"analytics": true,
	"redirect": true,
}

// validateCustomShortCode validates a custom short code
func (s *urlService) validateCustomShortCode(shortCode string) error {
	// Check length (min 3, max 20 characters)
	if len(shortCode) < 3 {
		return fmt.Errorf("short code must be at least 3 characters long")
	}
	if len(shortCode) > 20 {
		return fmt.Errorf("short code must be at most 20 characters long")
	}

	// Check format: only alphanumeric characters and hyphens/underscores
	matched, err := regexp.MatchString("^[a-zA-Z0-9_-]+$", shortCode)
	if err != nil {
		return fmt.Errorf("failed to validate short code format: %w", err)
	}
	if !matched {
		return fmt.Errorf("short code can only contain letters, numbers, hyphens, and underscores")
	}

	// Check if it's a reserved word (case-insensitive)
	if reservedCodes[strings.ToLower(shortCode)] {
		return fmt.Errorf("short code '%s' is reserved and cannot be used", shortCode)
	}

	return nil
}

// generateShortCode generates a random 8-character short code
func (s *urlService) generateShortCode() (string, error) {
	// Generate 6 bytes of random data
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base64 URL-safe string and take first 8 characters
	encoded := base64.URLEncoding.EncodeToString(bytes)
	return encoded[:8], nil
}

// checkShortCodeAvailability checks if a short code is available using Redis cache first
func (s *urlService) checkShortCodeAvailability(shortCode string) (bool, error) {
	// Check Redis cache first (if available)
	if s.cache != nil {
		cacheKey := fmt.Sprintf("shortcode:exists:%s", shortCode)
		exists, err := s.cache.Exists(s.ctx, cacheKey)
		if err == nil && exists {
			// Key exists in cache, check if it's marked as taken
			val, err := s.cache.Get(s.ctx, cacheKey)
			if err == nil && val == "taken" {
				return false, nil
			}
		}
	}

	// If not in cache or cache miss, check database
	_, err := s.repo.FindByShortCode(shortCode)
	if err != nil {
		// If error is "URL not found", the code is available
		if err.Error() == "URL not found or expired" {
			// Cache that it's available (with short TTL to allow for race conditions)
			if s.cache != nil {
				cacheKey := fmt.Sprintf("shortcode:exists:%s", shortCode)
				s.cache.Set(s.ctx, cacheKey, "available", 30*time.Second)
			}
			return true, nil
		}
		// Other errors (like database errors) should be returned
		return false, err
	}
	// If no error, URL exists, so code is not available
	// Cache that it's taken (with longer TTL)
	if s.cache != nil {
		cacheKey := fmt.Sprintf("shortcode:exists:%s", shortCode)
		s.cache.Set(s.ctx, cacheKey, "taken", 1*time.Hour)
	}
	return false, nil
}

// CreateShortURL creates a new short URL
func (s *urlService) CreateShortURL(req *models.CreateURLRequest, userID *string, baseURL string) (*models.CreateURLResponse, error) {
	// Validate expiration time if provided
	// Allow a 2-second buffer to account for network latency and processing time
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now().Add(-2*time.Second)) {
		return nil, fmt.Errorf("expiration time cannot be in the past")
	}

	var shortCode string
	var err error

	// If custom short code is provided, validate and use it
	if req.ShortCode != nil && *req.ShortCode != "" {
		customCode := strings.TrimSpace(*req.ShortCode)
		
		// Validate custom code
		if err := s.validateCustomShortCode(customCode); err != nil {
			return nil, err
		}

		// Check availability
		available, err := s.checkShortCodeAvailability(customCode)
		if err != nil {
			return nil, fmt.Errorf("failed to check short code availability: %w", err)
		}
		if !available {
			return nil, fmt.Errorf("short code '%s' is already taken", customCode)
		}

		shortCode = customCode
	} else {
		// Generate unique short code (retry if collision occurs)
		maxAttempts := 10
		for i := 0; i < maxAttempts; i++ {
			shortCode, err = s.generateShortCode()
			if err != nil {
				return nil, err
			}

			// Check if generated code is available (should be, but check anyway)
			available, err := s.checkShortCodeAvailability(shortCode)
			if err != nil {
				return nil, fmt.Errorf("failed to check short code availability: %w", err)
			}
			if available {
				break
			}

			// If not available and we've exhausted attempts, return error
			if i == maxAttempts-1 {
				return nil, fmt.Errorf("failed to generate unique short code after %d attempts", maxAttempts)
			}
		}
	}

	// Create the URL with the determined short code
	url, err := s.repo.Create(shortCode, req.URL, userID, req.ExpiresAt)
	if err != nil {
		// Check if it's a unique constraint violation (shouldn't happen if we checked, but handle it)
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			// Mark as taken in cache
			if s.cache != nil {
				cacheKey := fmt.Sprintf("shortcode:exists:%s", shortCode)
				s.cache.Set(s.ctx, cacheKey, "taken", 1*time.Hour)
			}
			return nil, fmt.Errorf("short code '%s' is already taken", shortCode)
		}
		return nil, fmt.Errorf("failed to create URL: %w", err)
	}

	// Mark as taken in cache and cache the URL lookup
	if s.cache != nil {
		cacheKey := fmt.Sprintf("shortcode:exists:%s", shortCode)
		s.cache.Set(s.ctx, cacheKey, "taken", 1*time.Hour)

		// Cache the URL lookup
		urlCacheKey := fmt.Sprintf("url:%s", shortCode)
		urlCacheData := map[string]interface{}{
			"original_url": url.OriginalURL,
			"expires_at":   url.ExpiresAt,
		}
		s.cache.SetJSON(s.ctx, urlCacheKey, urlCacheData, 1*time.Hour)
	}

	// Success! Convert entity to response DTO
	return &models.CreateURLResponse{
		ShortCode:   url.ShortCode,
		OriginalURL: url.OriginalURL,
		ShortURL:    fmt.Sprintf("%s/%s", baseURL, url.ShortCode),
		ExpiresAt:   url.ExpiresAt,
		CreatedAt:   url.CreatedAt,
	}, nil
}

// GetOriginalURL retrieves the original URL and increments click count
func (s *urlService) GetOriginalURL(shortCode string) (string, error) {
	// Try cache first (if available)
	if s.cache != nil {
		urlCacheKey := fmt.Sprintf("url:%s", shortCode)
		var cachedData map[string]interface{}
		err := s.cache.GetJSON(s.ctx, urlCacheKey, &cachedData)
		if err == nil && cachedData != nil {
			originalURL, ok := cachedData["original_url"].(string)
			if ok && originalURL != "" {
				// Check if expired
				if expiresAtVal, exists := cachedData["expires_at"]; exists && expiresAtVal != nil {
					if expiresAtStr, ok := expiresAtVal.(string); ok && expiresAtStr != "" {
						expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
						if err == nil && expiresAt.Before(time.Now()) {
							// Expired, remove from cache and check DB
							s.cache.Delete(s.ctx, urlCacheKey)
						} else {
							// Not expired, return cached URL
							go func() {
								if err := s.repo.IncrementClickCount(shortCode); err != nil {
									fmt.Printf("Warning: failed to increment click count for %s: %v\n", shortCode, err)
								}
							}()
							return originalURL, nil
						}
					}
				} else {
					// No expiration, return cached URL
					go func() {
						if err := s.repo.IncrementClickCount(shortCode); err != nil {
							fmt.Printf("Warning: failed to increment click count for %s: %v\n", shortCode, err)
						}
					}()
					return originalURL, nil
				}
			}
		}
	}

	// Cache miss or expired, get from database
	url, err := s.repo.FindByShortCode(shortCode)
	if err != nil {
		return "", err
	}

	// Cache the result
	if s.cache != nil {
		urlCacheKey := fmt.Sprintf("url:%s", shortCode)
		urlCacheData := map[string]interface{}{
			"original_url": url.OriginalURL,
			"expires_at":   url.ExpiresAt,
		}
		s.cache.SetJSON(s.ctx, urlCacheKey, urlCacheData, 1*time.Hour)
	}

	// Increment click count synchronously
	// This is a fast operation and ensures clicks are logged reliably
	if err := s.repo.IncrementClickCount(shortCode); err != nil {
		// Log error but don't fail the redirect
		fmt.Printf("Warning: failed to increment click count for %s: %v\n", shortCode, err)
	}

	return url.OriginalURL, nil
}

// GetURLStats retrieves statistics for a URL
func (s *urlService) GetURLStats(shortCode string, userID *string) (*models.URLStatsResponse, error) {
	url, err := s.repo.GetStats(shortCode, userID)
	if err != nil {
		return nil, err
	}

	return &models.URLStatsResponse{
		ShortCode:   url.ShortCode,
		OriginalURL: url.OriginalURL,
		ClickCount:  url.ClickCount,
		CreatedAt:   url.CreatedAt,
		ExpiresAt:   url.ExpiresAt,
	}, nil
}

// DeleteURL deletes a URL by short code
func (s *urlService) DeleteURL(shortCode string, userID *string) error {
	err := s.repo.Delete(shortCode, userID)
	if err == nil && s.cache != nil {
		// Invalidate cache
		urlCacheKey := fmt.Sprintf("url:%s", shortCode)
		s.cache.Delete(s.ctx, urlCacheKey)
		cacheKey := fmt.Sprintf("shortcode:exists:%s", shortCode)
		s.cache.Delete(s.ctx, cacheKey)
	}
	return err
}

// UpdateExpiresAt updates the expiration date for a URL
func (s *urlService) UpdateExpiresAt(shortCode string, userID *string, expiresAt *time.Time) error {
	// Validate expiration time if provided
	// Allow a 2-second buffer to account for network latency and processing time
	if expiresAt != nil && expiresAt.Before(time.Now().Add(-2*time.Second)) {
		return fmt.Errorf("expiration time cannot be in the past")
	}

	return s.repo.UpdateExpiresAt(shortCode, userID, expiresAt)
}

// GetUserURLs retrieves all URLs for a specific user
func (s *urlService) GetUserURLs(userID string) ([]*models.URLStatsResponse, error) {
	urls, err := s.repo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	responses := make([]*models.URLStatsResponse, len(urls))
	for i, url := range urls {
		responses[i] = &models.URLStatsResponse{
			ShortCode:   url.ShortCode,
			OriginalURL: url.OriginalURL,
			ClickCount:  url.ClickCount,
			CreatedAt:   url.CreatedAt,
			ExpiresAt:   url.ExpiresAt,
		}
	}

	return responses, nil
}

// GetClickAnalytics retrieves click analytics for a URL
func (s *urlService) GetClickAnalytics(shortCode string, userID *string, hours int) ([]map[string]interface{}, error) {
	// First verify the URL exists and user has access
	url, err := s.repo.GetStats(shortCode, userID)
	if err != nil {
		return nil, err
	}

	// Get analytics
	return s.repo.GetClickAnalytics(url.ID, hours)
}
