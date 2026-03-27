package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"shmanki/internal/platform/response"
	"shmanki/internal/platform/token"
)

type contextKey string

const userIDContextKey contextKey = "userID"

func Auth(manager *token.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			if authorization == "" {
				response.WriteError(w, http.StatusUnauthorized, "missing authorization header", "UNAUTHORIZED")
				return
			}

			parts := strings.SplitN(authorization, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.WriteError(w, http.StatusUnauthorized, "invalid authorization header", "UNAUTHORIZED")
				return
			}

			userID, err := manager.Parse(parts[1])
			if err != nil {
				response.WriteError(w, http.StatusUnauthorized, "invalid token", "UNAUTHORIZED")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) uuid.UUID {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}

	return userID
}
