package handler

import (
	"encoding/json"
	"errors"
	"hornetSecurity/internal/repository"
	"hornetSecurity/internal/service"
	"io"
	"net/http"
	"os"
	"strconv"
)

func (e *apiError) Error() string { return e.message }

func badRequest(message string) error {
	return &apiError{status: http.StatusBadRequest, message: message}
}

func (h *documentHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *apiError
	switch {
	case errors.As(err, &apiErr):
		h.writeJSON(w, r, apiErr.status, errorResponse{Error: apiErr.message})
	case errors.Is(err, service.ErrInvalidDocument):
		h.writeJSON(w, r, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, repository.ErrNotFound):
		h.writeJSON(w, r, http.StatusNotFound, errorResponse{Error: "document not found"})
	default:
		h.logger.ErrorContext(r.Context(), "request failed",
			"method", r.Method, "path", r.URL.Path, "error", err)
		h.writeJSON(w, r, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func parseIDFromPath(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, badRequest("id must be a positive integer")
	}
	return id, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return &apiError{status: http.StatusRequestEntityTooLarge, message: "request body too large"}
		}
		return badRequest("invalid JSON body")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return badRequest("body must contain a single JSON object")
	}
	return nil
}

func (h *documentHandler) writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.ErrorContext(r.Context(), "encode response",
			"method", r.Method, "path", r.URL.Path, "error", err)
	}
}

func GetPort() string {
	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}
	return addr
}
