package generate

import (
	"errors"
	"log"
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

func (h *Handler) Suggest(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	var req SuggestRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	result, err := h.service.Suggest(r.Context(), userID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	var req SaveRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	result, err := h.service.Save(r.Context(), userID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	var req EditRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	result, err := h.service.Edit(r.Context(), userID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrGenerationUnavailable):
		response.WriteError(w, http.StatusNotImplemented, err.Error(), "INTERNAL_ERROR")
	case errors.Is(err, ErrInvalidRequest):
		response.WriteError(w, http.StatusBadRequest, err.Error(), "INVALID_REQUEST")
	case errors.Is(err, ErrInvalidSuggestion):
		response.WriteError(w, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
	case errors.Is(err, ErrDeckNotFound):
		response.WriteError(w, http.StatusNotFound, err.Error(), "NOT_FOUND")
	default:
		log.Printf("[generate] internal error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}
