package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mike/fitassist/internal/config"
	"github.com/mike/fitassist/internal/model"
	"github.com/mike/fitassist/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotActive      = errors.New("user is not active")
)

type AuthService struct {
	userRepo *repository.UserRepository
	security config.SecurityConfig
}

func NewAuthService(userRepo *repository.UserRepository, security config.SecurityConfig) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		security: security,
	}
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthService) Register(ctx context.Context, username, password, email string) (*model.User, error) {
	existing, _ := s.userRepo.GetByUsername(ctx, username)
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         "user",
		IsActive:     true,
	}
	if email != "" {
		user.Email = &email
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*TokenPair, *model.User, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, ErrUserNotActive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrUserNotActive
	}

	return s.generateTokens(user)
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.security.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidCredentials
	}

	return claims, nil
}

func (s *AuthService) generateTokens(user *model.User) (*TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(15 * time.Minute)

	accessClaims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID,
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString([]byte(s.security.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	refreshClaims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID,
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString([]byte(s.security.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("signing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresAt:    accessExp.Unix(),
	}, nil
}

// EnsureAdmin creates the initial admin user if no users exist.
func EnsureAdmin(ctx context.Context, userRepo *repository.UserRepository, cfg config.AdminConfig) {
	count, err := userRepo.Count(ctx)
	if err != nil {
		slog.Error("checking user count", "error", err)
		return
	}
	if count > 0 {
		return
	}

	if cfg.InitialUsername == "" || cfg.InitialPassword == "" {
		slog.Warn("no admin credentials configured, skipping initial admin creation")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.InitialPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hashing admin password", "error", err)
		return
	}

	admin := &model.User{
		Username:     cfg.InitialUsername,
		PasswordHash: string(hash),
		Role:         "admin",
		IsActive:     true,
	}

	if err := userRepo.Create(ctx, admin); err != nil {
		slog.Error("creating initial admin", "error", err)
		return
	}

	slog.Info("initial admin user created", "username", cfg.InitialUsername)
}
