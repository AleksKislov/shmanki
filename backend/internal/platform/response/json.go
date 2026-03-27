package response

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if payload == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf("encode response: %v", err), http.StatusInternalServerError)
	}
}

func WriteError(w http.ResponseWriter, status int, message string, code string) {
	WriteJSON(w, status, ErrorBody{Error: message, Code: code})
}

func DecodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	return nil
}
