package review

import (
	"errors"
	"net/http"
	"strconv"
	"time"

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

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.GetSession(r.Context(), userID, limit)
	if err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	var req ReviewRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	result, err := h.service.SubmitReview(r.Context(), userID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) GetDeckStats(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}

	stats, err := h.service.GetDeckStats(r.Context(), userID, deckID)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, stats)
}

func (h *Handler) GetCardStats(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	cardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid card id", "INVALID_REQUEST")
		return
	}

	history, err := h.service.GetCardHistory(r.Context(), userID, cardID)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, history)
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrReviewCardNotFound):
		response.WriteError(w, http.StatusNotFound, err.Error(), "NOT_FOUND")
	case errors.Is(err, ErrInvalidSubmission):
		response.WriteError(w, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
	default:
		response.WriteError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
