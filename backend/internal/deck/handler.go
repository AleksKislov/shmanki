package deck

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	platformmiddleware "shmanki/internal/platform/middleware"
	"shmanki/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	items, err := h.service.List(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	var req CreateDeckRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	item, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}

	item, err := h.service.Get(r.Context(), userID, deckID)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}

	var req UpdateDeckRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	item, err := h.service.Update(r.Context(), userID, deckID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}

	if err := h.service.Delete(r.Context(), userID, deckID); err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrDeckNotFound):
		response.WriteError(w, http.StatusNotFound, err.Error(), "NOT_FOUND")
	case errors.Is(err, ErrInvalidDeck), errors.Is(err, ErrInvalidLang):
		response.WriteError(w, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
	default:
		response.WriteError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}
