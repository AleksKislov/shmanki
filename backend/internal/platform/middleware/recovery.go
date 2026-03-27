package middleware

import (
	"log"
	"net/http"

	"shmanki/internal/platform/response"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic recovered: %v", recovered)
				response.WriteError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
