package repository

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"shortly-be/internal/entities"
)

// URLRepository defines the interface for URL database operations
type URLRepository interface {
	Create(shortCode, originalURL string, userID *string, expiresAt *time.Time) (*entities.URL, error)
	FindByShortCode(shortCode string) (*entities.URL, error)
	IncrementClickCount(shortCode string) error
	Delete(shortCode string, userID *string) error
	UpdateExpiresAt(shortCode string, userID *string, expiresAt *time.Time) error
	GetStats(shortCode string, userID *string) (*entities.URL, error)
	GetByUserID(userID string) ([]*entities.URL, error)
	GetClickAnalytics(urlID string, hours int) ([]map[string]interface{}, error)
}

type urlRepository struct {
	db *sql.DB
}

// NewURLRepository creates a new URL repository
func NewURLRepository(db *sql.DB) URLRepository {
	return &urlRepository{db: db}
}

// Create inserts a new URL into the database
func (r *urlRepository) Create(shortCode, originalURL string, userID *string, expiresAt *time.Time) (*entities.URL, error) {
	// Ensure expiresAt is stored in UTC
	var expiresAtValue interface{}
	if expiresAt != nil {
		// Convert to UTC and use explicit UTC timestamp in SQL
		utcTime := expiresAt.UTC()
		expiresAtValue = utcTime
	} else {
		expiresAtValue = nil
	}

	query := `
		INSERT INTO urls (short_code, original_url, user_id, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, short_code, original_url, user_id, click_count, created_at, expires_at
	`

	var url entities.URL
	err := r.db.QueryRow(query, shortCode, originalURL, userID, expiresAtValue).Scan(
		&url.ID,
		&url.ShortCode,
		&url.OriginalURL,
		&url.UserID,
		&url.ClickCount,
		&url.CreatedAt,
		&url.ExpiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create URL: %w", err)
	}

	return &url, nil
}

// FindByShortCode finds a URL by its short code (only if not expired)
func (r *urlRepository) FindByShortCode(shortCode string) (*entities.URL, error) {
	query := `
		SELECT id, short_code, original_url, user_id, click_count, created_at, expires_at
		FROM urls
		WHERE short_code = $1
		AND (expires_at IS NULL OR expires_at > (NOW() AT TIME ZONE 'UTC'))
	`

	var url entities.URL
	err := r.db.QueryRow(query, shortCode).Scan(
		&url.ID,
		&url.ShortCode,
		&url.OriginalURL,
		&url.UserID,
		&url.ClickCount,
		&url.CreatedAt,
		&url.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("URL not found or expired")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find URL: %w", err)
	}

	return &url, nil
}

// IncrementClickCount increments the click count for a URL and logs the click
func (r *urlRepository) IncrementClickCount(shortCode string) error {
	// First, get the URL ID
	var urlID string
	err := r.db.QueryRow("SELECT id FROM urls WHERE short_code = $1", shortCode).Scan(&urlID)
	if err != nil {
		return fmt.Errorf("failed to find URL: %w", err)
	}

	// Update click count
	_, err = r.db.Exec(`
		UPDATE urls
		SET click_count = click_count + 1
		WHERE short_code = $1
	`, shortCode)
	if err != nil {
		return fmt.Errorf("failed to increment click count: %w", err)
	}

	// Log the click with timestamp in UTC
	_, err = r.db.Exec(`
		INSERT INTO url_clicks (url_id, clicked_at)
		VALUES ($1, (NOW() AT TIME ZONE 'UTC'))
	`, urlID)
	if err != nil {
		// Log the error with more context
		log.Printf("ERROR: Failed to insert click for url_id=%s, short_code=%s: %v", urlID, shortCode, err)
		// Check if it's a table doesn't exist error
		if err.Error() == "pq: relation \"url_clicks\" does not exist" {
			log.Printf("ERROR: url_clicks table does not exist! Please run migrations.")
		}
		return fmt.Errorf("failed to log click: %w", err)
	}

	return nil
}

// Delete removes a URL from the database (only if user owns it or userID is nil)
func (r *urlRepository) Delete(shortCode string, userID *string) error {
	var query string
	var args []interface{}

	if userID != nil {
		query = `DELETE FROM urls WHERE short_code = $1 AND user_id = $2`
		args = []interface{}{shortCode, *userID}
	} else {
		query = `DELETE FROM urls WHERE short_code = $1`
		args = []interface{}{shortCode}
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete URL: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("URL not found or you don't have permission to delete it")
	}

	return nil
}

// GetStats retrieves URL statistics (including expired URLs)
// If userID is provided, only returns stats if the URL belongs to that user
func (r *urlRepository) GetStats(shortCode string, userID *string) (*entities.URL, error) {
	var query string
	var args []interface{}

	if userID != nil {
		query = `
			SELECT id, short_code, original_url, user_id, click_count, created_at, expires_at
			FROM urls
			WHERE short_code = $1 AND user_id = $2
		`
		args = []interface{}{shortCode, *userID}
	} else {
		query = `
			SELECT id, short_code, original_url, user_id, click_count, created_at, expires_at
			FROM urls
			WHERE short_code = $1
		`
		args = []interface{}{shortCode}
	}

	var url entities.URL
	err := r.db.QueryRow(query, args...).Scan(
		&url.ID,
		&url.ShortCode,
		&url.OriginalURL,
		&url.UserID,
		&url.ClickCount,
		&url.CreatedAt,
		&url.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("URL not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &url, nil
}

// GetByUserID retrieves all URLs for a specific user
func (r *urlRepository) GetByUserID(userID string) ([]*entities.URL, error) {
	query := `
		SELECT id, short_code, original_url, user_id, click_count, created_at, expires_at
		FROM urls
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get URLs: %w", err)
	}
	defer rows.Close()

	var urls []*entities.URL
	for rows.Next() {
		var url entities.URL
		err := rows.Scan(
			&url.ID,
			&url.ShortCode,
			&url.OriginalURL,
			&url.UserID,
			&url.ClickCount,
			&url.CreatedAt,
			&url.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan URL: %w", err)
		}
		urls = append(urls, &url)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating URLs: %w", err)
	}

	return urls, nil
}

// GetClickAnalytics retrieves click analytics grouped by time intervals
func (r *urlRepository) GetClickAnalytics(urlID string, hours int) ([]map[string]interface{}, error) {
	// DATE_TRUNC only accepts: minute, hour, day, week, month, etc.
	// For custom intervals, we need to use a different approach
	var query string
	
	switch {
	case hours <= 6:
		// Group by 10 minutes (all times in UTC)
		query = fmt.Sprintf(`
			SELECT 
				(DATE_TRUNC('hour', clicked_at AT TIME ZONE 'UTC') + 
				INTERVAL '10 minutes' * FLOOR(EXTRACT(MINUTE FROM clicked_at AT TIME ZONE 'UTC') / 10)) AT TIME ZONE 'UTC' as time_bucket,
				COUNT(*) as click_count
			FROM url_clicks
			WHERE url_id = $1
			AND clicked_at >= (NOW() AT TIME ZONE 'UTC') - INTERVAL '%d hours'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, hours)
	case hours <= 12:
		// Group by 30 minutes (all times in UTC)
		query = fmt.Sprintf(`
			SELECT 
				(DATE_TRUNC('hour', clicked_at AT TIME ZONE 'UTC') + 
				INTERVAL '30 minutes' * FLOOR(EXTRACT(MINUTE FROM clicked_at AT TIME ZONE 'UTC') / 30)) AT TIME ZONE 'UTC' as time_bucket,
				COUNT(*) as click_count
			FROM url_clicks
			WHERE url_id = $1
			AND clicked_at >= (NOW() AT TIME ZONE 'UTC') - INTERVAL '%d hours'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, hours)
	case hours <= 24:
		// Group by 1 hour (all times in UTC)
		query = fmt.Sprintf(`
			SELECT 
				DATE_TRUNC('hour', clicked_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' as time_bucket,
				COUNT(*) as click_count
			FROM url_clicks
			WHERE url_id = $1
			AND clicked_at >= (NOW() AT TIME ZONE 'UTC') - INTERVAL '%d hours'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, hours)
	case hours <= 72: // 3 days
		// Group by 6 hours - round to nearest 6 hour block (all times in UTC)
		query = fmt.Sprintf(`
			SELECT 
				(DATE_TRUNC('day', clicked_at AT TIME ZONE 'UTC') + 
				INTERVAL '6 hours' * FLOOR(EXTRACT(HOUR FROM clicked_at AT TIME ZONE 'UTC') / 6)) AT TIME ZONE 'UTC' as time_bucket,
				COUNT(*) as click_count
			FROM url_clicks
			WHERE url_id = $1
			AND clicked_at >= (NOW() AT TIME ZONE 'UTC') - INTERVAL '%d hours'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, hours)
	default: // 7 days, 14 days, 30 days
		// Group by 1 day (all times in UTC)
		query = fmt.Sprintf(`
			SELECT 
				DATE_TRUNC('day', clicked_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' as time_bucket,
				COUNT(*) as click_count
			FROM url_clicks
			WHERE url_id = $1
			AND clicked_at >= (NOW() AT TIME ZONE 'UTC') - INTERVAL '%d hours'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, hours)
	}

	rows, err := r.db.Query(query, urlID)
	if err != nil {
		return nil, fmt.Errorf("failed to get click analytics: %w", err)
	}
	defer rows.Close()

	var analytics []map[string]interface{}
	for rows.Next() {
		var timeBucket time.Time
		var count int
		if err := rows.Scan(&timeBucket, &count); err != nil {
			return nil, fmt.Errorf("failed to scan analytics: %w", err)
		}

		analytics = append(analytics, map[string]interface{}{
			"time":  timeBucket,
			"count": count,
		})
	}

	return analytics, nil
}

// UpdateExpiresAt updates the expiration date for a URL (only if user owns it)
func (r *urlRepository) UpdateExpiresAt(shortCode string, userID *string, expiresAt *time.Time) error {
	if userID == nil {
		return fmt.Errorf("user ID required")
	}

	// Ensure expiresAt is stored in UTC
	var expiresAtValue interface{}
	if expiresAt != nil {
		utcTime := expiresAt.UTC()
		expiresAtValue = utcTime
	} else {
		expiresAtValue = nil
	}

	query := `
		UPDATE urls
		SET expires_at = $1
		WHERE short_code = $2 AND user_id = $3
	`

	result, err := r.db.Exec(query, expiresAtValue, shortCode, *userID)
	if err != nil {
		return fmt.Errorf("failed to update URL: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("URL not found or you don't have permission to update it")
	}

	return nil
}

