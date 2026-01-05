package controllers

import (
	"net/http"
	"strconv"
	"time"

	"shortly-be/internal/models"
	"shortly-be/internal/service"

	"github.com/gin-gonic/gin"
)

type ShortenerController struct {
	urlService service.URLService
	baseURL    string
}

func NewShortenerController(urlService service.URLService, baseURL string) *ShortenerController {
	return &ShortenerController{
		urlService: urlService,
		baseURL:    baseURL,
	}
}

// CreateShortURL handles POST /api/v1/shorten
func (sc *ShortenerController) CreateShortURL(c *gin.Context) {
	var req models.CreateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get user ID from JWT context (set by auth middleware) - UUID string
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		c.Abort()
		return
	}
	userID := userIDStr.(string)

	// Ensure expiresAt is in UTC
	if req.ExpiresAt != nil {
		utcTime := req.ExpiresAt.UTC()
		req.ExpiresAt = &utcTime
	}

	response, err := sc.urlService.CreateShortURL(&req, &userID, sc.baseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// RedirectToURL handles GET /:shortCode - redirects to original URL
func (sc *ShortenerController) RedirectToURL(c *gin.Context) {
	shortCode := c.Param("shortCode")

	originalURL, err := sc.urlService.GetOriginalURL(shortCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Short URL not found or expired",
		})
		return
	}

	// Redirect with 301 (Moved Permanently)
	c.Redirect(http.StatusMovedPermanently, originalURL)
}

// GetOriginalURLPublic handles GET /api/v1/redirect/:shortCode - returns original URL as JSON (public, no auth)
func (sc *ShortenerController) GetOriginalURLPublic(c *gin.Context) {
	shortCode := c.Param("shortCode")

	originalURL, err := sc.urlService.GetOriginalURL(shortCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Short URL not found or expired",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"original_url": originalURL,
	})
}

// GetURLStats handles GET /api/v1/url/:shortCode - returns URL statistics
func (sc *ShortenerController) GetURLStats(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Get user ID from JWT context (set by auth middleware) - UUID string
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		c.Abort()
		return
	}
	userID := userIDStr.(string)

	stats, err := sc.urlService.GetURLStats(shortCode, &userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "URL not found",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetClickAnalytics handles GET /api/v1/url/:shortCode/analytics - returns click analytics
func (sc *ShortenerController) GetClickAnalytics(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Get user ID from JWT context (set by auth middleware) - UUID string
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		c.Abort()
		return
	}
	userID := userIDStr.(string)

	// Get hours parameter (default to 24)
	hours := 24
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if parsedHours, err := strconv.Atoi(hoursStr); err == nil && parsedHours > 0 {
			hours = parsedHours
		}
	}

	analytics, err := sc.urlService.GetClickAnalytics(shortCode, &userID, hours)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

// DeleteURL handles DELETE /api/v1/url/:shortCode - deletes a URL
func (sc *ShortenerController) DeleteURL(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Get user ID from JWT context (set by auth middleware) - UUID string
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		c.Abort()
		return
	}
	userID := userIDStr.(string)

	err := sc.urlService.DeleteURL(shortCode, &userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "URL deleted successfully",
	})
}

// GetUserURLs handles GET /api/v1/urls - returns all URLs for the authenticated user
func (sc *ShortenerController) GetUserURLs(c *gin.Context) {
	// Get user ID from JWT context (set by auth middleware) - UUID string
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		c.Abort()
		return
	}
	userID := userIDStr.(string)

	urls, err := sc.urlService.GetUserURLs(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, urls)
}

// UpdateURLExpiresAt handles PATCH /api/v1/url/:shortCode - updates expiration date
func (sc *ShortenerController) UpdateURLExpiresAt(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Get user ID from JWT context (set by auth middleware) - UUID string
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		c.Abort()
		return
	}
	userID := userIDStr.(string)

	var req struct {
		ExpiresAt *string `json:"expires_at"` // ISO 8601 string or null
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid date format. Use ISO 8601 format (e.g., 2024-12-31T23:59:59Z)",
			})
			return
		}
		// Ensure the time is in UTC
		utcTime := parsed.UTC()
		expiresAt = &utcTime
	}

	err := sc.urlService.UpdateExpiresAt(shortCode, &userID, expiresAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "URL expiration updated successfully",
	})
}

