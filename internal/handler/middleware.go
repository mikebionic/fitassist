package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mike/fitassist/internal/service"
)

type contextKey string

const (
	CtxUserID   contextKey = "user_id"
	CtxUsername  contextKey = "username"
	CtxUserRole contextKey = "user_role"
	ctxJWTSecret contextKey = "jwt_secret"
)

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "authorization header required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}

			tokenStr := parts[1]

			token, err := jwt.ParseWithClaims(tokenStr, &service.Claims{}, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			claims, ok := token.Claims.(*service.Claims)
			if !ok || !token.Valid {
				writeError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, CtxUserID, claims.UserID)
			ctx = context.WithValue(ctx, CtxUsername, claims.Username)
			ctx = context.WithValue(ctx, CtxUserRole, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(CtxUserRole).(string)
		if !ok || role != "admin" {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserID(r *http.Request) string {
	if v, ok := r.Context().Value(CtxUserID).(string); ok {
		return v
	}
	return ""
}

// JWTSecretMiddleware injects the JWT secret into the request context
// so WebSocket handlers can validate tokens without the auth middleware.
func JWTSecretMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxJWTSecret, jwtSecret)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// parseJWTUserID extracts the user ID from a JWT token string.
func parseJWTUserID(tokenStr, jwtSecret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &service.Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*service.Claims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token claims")
	}

	return claims.UserID, nil
}
