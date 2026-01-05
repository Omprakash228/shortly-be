package service

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"shortly-be/internal/jwt"
	"shortly-be/internal/models"
	"shortly-be/internal/repository"
)

// AuthService defines the interface for authentication business logic
type AuthService interface {
	Register(req *models.RegisterRequest) (*models.RegisterResponse, error)
	Login(req *models.LoginRequest) (*models.AuthResponse, error)
}

type authService struct {
	userRepo  repository.UserRepository
	jwtService *jwt.JWTService
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo repository.UserRepository, jwtService *jwt.JWTService) AuthService {
	return &authService{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

// Register creates a new user account
func (s *authService) Register(req *models.RegisterRequest) (*models.RegisterResponse, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.FindByEmail(req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.userRepo.Create(req.Email, string(hashedPassword), req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate JWT token for automatic login after registration
	token, err := s.jwtService.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.RegisterResponse{
		Message: "User registered successfully",
		User: models.AuthResponse{
			UserID:    user.ID,
			Email:     user.Email,
			Name:      user.Name,
			CreatedAt: user.CreatedAt,
			Token:     token,
		},
	}, nil
}

// Login authenticates a user and returns user info with JWT token
func (s *authService) Login(req *models.LoginRequest) (*models.AuthResponse, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.AuthResponse{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
		Token:     token,
	}, nil
}

