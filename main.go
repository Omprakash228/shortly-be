package main

import (
	"log"
	"time"

	"shortly-be/internal/cache"
	"shortly-be/internal/config"
	"shortly-be/internal/controllers"
	"shortly-be/internal/database"
	"shortly-be/internal/jwt"
	"shortly-be/internal/middleware"
	"shortly-be/internal/repository"
	"shortly-be/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close() // Close connection when program exits

 	// Run database migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Redis cache (optional - continue if Redis is unavailable)
	var cacheClient cache.Cache
	cacheClient, err = cache.NewRedisCache(cfg.RedisURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis (%v). Continuing without cache.", err)
		cacheClient = nil
	} else {
		log.Println("Connected to Redis cache")
	}

	// Initialize repositories
	urlRepo := repository.NewURLRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Initialize JWT service
	jwtService := jwt.NewJWTService(
		cfg.JWTSecret,
		time.Duration(cfg.JWTTTL)*time.Hour,
	)

	// Initialize services
	urlService := service.NewURLService(urlRepo, cacheClient)
	authService := service.NewAuthService(userRepo, jwtService)

	// Initialize controllers
	shortenerController := controllers.NewShortenerController(urlService, cfg.BaseURL)
	authController := controllers.NewAuthController(authService)
	qrcodeController := controllers.NewQRCodeController(cfg.FrontendURL)

	// Initialize rate limiters
	generalRateLimiter := middleware.NewRateLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst)
	authRateLimiter := middleware.NewRateLimiter(rate.Limit(cfg.RateLimitAuthRPS), cfg.RateLimitAuthBurst)
	shortenRateLimiter := middleware.NewRateLimiter(rate.Limit(cfg.RateLimitShortenRPS), cfg.RateLimitShortenBurst)
	redirectRateLimiter := middleware.NewRateLimiter(rate.Limit(30.0), 60) // More lenient for redirects (30 req/s, burst 60)

	// Create a Gin router
	router := gin.Default()

	// Health check endpoint (no rate limiting)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Redirect endpoint with rate limiting
	router.GET("/:shortCode", redirectRateLimiter.LimitMiddleware(), shortenerController.RedirectToURL)

	// API v1 routes group with general rate limiting
	api := router.Group("/api/v1")
	api.Use(generalRateLimiter.LimitMiddleware())
	{
		// Auth routes with stricter rate limiting
		auth := api.Group("/auth")
		auth.Use(authRateLimiter.LimitMiddleware())
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
		}

		// Protected routes - require JWT authentication
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(jwtService))
		{
			// URL shortening with stricter rate limiting
			protected.POST("/shorten", shortenRateLimiter.LimitMiddleware(), shortenerController.CreateShortURL)
			
			// Other URL routes (use general rate limiting from group)
			protected.GET("/urls", shortenerController.GetUserURLs)
			protected.GET("/url/:shortCode", shortenerController.GetURLStats)
			protected.GET("/url/:shortCode/analytics", shortenerController.GetClickAnalytics)
			protected.PATCH("/url/:shortCode", shortenerController.UpdateURLExpiresAt)
			protected.DELETE("/url/:shortCode", shortenerController.DeleteURL)
		}
		
		// Public redirect endpoint with lenient rate limiting (same as direct redirect)
		api.GET("/redirect/:shortCode", redirectRateLimiter.LimitMiddleware(), shortenerController.GetOriginalURLPublic)
		
		// QR Code generation
		api.GET("/qrcode/:shortCode", qrcodeController.GenerateQRCode)
	}

	// Start the server on port 8080
	log.Println("Server starting on http://localhost:8080")
	router.Run(":8080")
}

