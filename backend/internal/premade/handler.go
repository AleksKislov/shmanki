package premade

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	platformmiddleware "shmanki/internal/platform/middleware"
	"shmanki/internal/platform/response"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	minRating, _ := strconv.Atoi(r.URL.Query().Get("min_rating"))
	items, err := h.service.List(r.Context(), ListFilters{
		Source:    r.URL.Query().Get("source"),
		Category:  r.URL.Query().Get("category"),
		Language:  r.URL.Query().Get("language"),
		MinRating: minRating,
		Sort:      r.URL.Query().Get("sort"),
		UserID:    userID,
	})
	if err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Categories(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	deckID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}
	userID := platformmiddleware.UserIDFromContext(r.Context())
	item, err := h.service.Get(r.Context(), deckID, userID)
	if err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	deckID, err := uuid.Parse(chi.URLParam(r, "deckId"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}
	var req PublishDeckRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}
	userID := platformmiddleware.UserIDFromContext(r.Context())
	premadeID, err := h.service.Publish(r.Context(), userID, deckID, req)
	if err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, map[string]string{"premadeDeckId": premadeID.String()})
}

func (h *Handler) Clone(w http.ResponseWriter, r *http.Request) {
	premadeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}
	userID := platformmiddleware.UserIDFromContext(r.Context())
	newDeckID, err := h.service.Clone(r.Context(), userID, premadeID)
	if err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, map[string]string{"deckId": newDeckID.String()})
}

func (h *Handler) Rate(w http.ResponseWriter, r *http.Request) {
	premadeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}
	var req RateDeckRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}
	userID := platformmiddleware.UserIDFromContext(r.Context())
	if err := h.service.Rate(r.Context(), userID, premadeID, req); err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Unrate(w http.ResponseWriter, r *http.Request) {
	premadeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}
	userID := platformmiddleware.UserIDFromContext(r.Context())
	if err := h.service.RemoveRating(r.Context(), userID, premadeID); err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	premadeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}
	userID := platformmiddleware.UserIDFromContext(r.Context())
	if err := h.service.Delete(r.Context(), userID, premadeID); err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.AdminList(r.Context(), r.URL.Query().Get("source"))
	if err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) AdminSetPublished(w http.ResponseWriter, r *http.Request) {
	premadeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}
	var req SetPublishedRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}
	if err := h.service.AdminSetPublished(r.Context(), premadeID, req.IsPublished); err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) AdminCreateOfficialFromDeck(w http.ResponseWriter, r *http.Request) {
	var req CreateOfficialFromDeckRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}
	premadeID, err := h.service.AdminCreateOfficialFromDeck(r.Context(), req)
	if err != nil {
		handleError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, map[string]string{"premadeDeckId": premadeID.String()})
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.WriteError(w, http.StatusNotFound, err.Error(), "NOT_FOUND")
	case errors.Is(err, ErrInvalidCategory), errors.Is(err, ErrInvalidRating), errors.Is(err, ErrAlreadyPublished):
		response.WriteError(w, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrRateOwnDeck):
		response.WriteError(w, http.StatusForbidden, err.Error(), "FORBIDDEN")
	default:
		response.WriteError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}
