package user

import (
	"errors"
	"net/http"

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

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmailTaken):
		response.WriteError(w, http.StatusConflict, err.Error(), "CONFLICT")
	case errors.Is(err, ErrInvalidCredentials):
		response.WriteError(w, http.StatusUnauthorized, err.Error(), "UNAUTHORIZED")
	case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrInvalidPassword):
		response.WriteError(w, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
	default:
		response.WriteError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}
