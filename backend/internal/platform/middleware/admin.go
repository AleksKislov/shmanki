package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"shmanki/internal/platform/response"
)

type userEmailGetter func(ctx context.Context, userID uuid.UUID) (string, error)

func RequireAdmin(adminEmails map[string]struct{}, getUserEmail userEmailGetter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			if userID == uuid.Nil {
				response.WriteError(w, http.StatusUnauthorized, "missing user", "UNAUTHORIZED")
				return
			}

			email, err := getUserEmail(r.Context(), userID)
			if err != nil {
				response.WriteError(w, http.StatusForbidden, "forbidden", "FORBIDDEN")
				return
			}

			if _, ok := adminEmails[strings.ToLower(strings.TrimSpace(email))]; !ok {
				response.WriteError(w, http.StatusForbidden, "forbidden", "FORBIDDEN")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
