package user

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

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	result, err := h.service.Register(r.Context(), req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	result, err := h.service.Login(r.Context(), req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) UpdatePreferredLanguage(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())

	var req UpdatePreferredLanguageRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if err := h.service.UpdatePreferredLanguage(r.Context(), userID, req); err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmailTaken):
		response.WriteError(w, http.StatusConflict, err.Error(), "CONFLICT")
	case errors.Is(err, ErrInvalidCredentials):
		response.WriteError(w, http.StatusUnauthorized, err.Error(), "UNAUTHORIZED")
	case errors.Is(err, ErrUserNotFound):
		response.WriteError(w, http.StatusNotFound, err.Error(), "NOT_FOUND")
	case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrInvalidPassword):
		response.WriteError(w, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
	case errors.Is(err, ErrInvalidLanguage):
		response.WriteError(w, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
	default:
		response.WriteError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}
