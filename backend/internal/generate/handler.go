package generate

import (
	"errors"
	"net/http"

	platformmiddleware "shmanki/internal/platform/middleware"
	"shmanki/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	var req Request
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	result, err := h.service.Generate(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrGenerationUnavailable):
			response.WriteError(w, http.StatusNotImplemented, err.Error(), "INTERNAL_ERROR")
		default:
			response.WriteError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
		}
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}
