package card

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

func (h *Handler) ListInfoObjects(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}

	objects, err := h.service.ListInfoObjects(r.Context(), userID, deckID)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, objects)
}

func (h *Handler) CreateInfoObject(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	deckID, err := uuid.Parse(chi.URLParam(r, "deckID"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid deck id", "INVALID_REQUEST")
		return
	}

	var req CreateInfoObjectRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	item, err := h.service.CreateInfoObject(r.Context(), userID, deckID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) GetInfoObject(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	objectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid object id", "INVALID_REQUEST")
		return
	}

	item, err := h.service.GetInfoObject(r.Context(), userID, objectID)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateInfoObject(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	objectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid object id", "INVALID_REQUEST")
		return
	}

	var req UpdateInfoObjectRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	item, err := h.service.UpdateInfoObject(r.Context(), userID, objectID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteInfoObject(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	objectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid object id", "INVALID_REQUEST")
		return
	}

	if err := h.service.DeleteInfoObject(r.Context(), userID, objectID); err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) CreateCard(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	objectID, err := uuid.Parse(chi.URLParam(r, "objectID"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid object id", "INVALID_REQUEST")
		return
	}

	var req CreateCardRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	item, err := h.service.CreateCard(r.Context(), userID, objectID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateCard(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	cardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid card id", "INVALID_REQUEST")
		return
	}

	var req UpdateCardRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	item, err := h.service.UpdateCard(r.Context(), userID, cardID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteCard(w http.ResponseWriter, r *http.Request) {
	userID := platformmiddleware.UserIDFromContext(r.Context())
	cardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid card id", "INVALID_REQUEST")
		return
	}

	if err := h.service.DeleteCard(r.Context(), userID, cardID); err != nil {
		handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInfoObjectNotFound), errors.Is(err, ErrCardNotFound):
		response.WriteError(w, http.StatusNotFound, err.Error(), "NOT_FOUND")
	case errors.Is(err, ErrInvalidInfoObject), errors.Is(err, ErrInvalidCard):
		response.WriteError(w, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
	default:
		response.WriteError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}
