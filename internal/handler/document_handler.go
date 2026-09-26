package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"hornetSecurity/internal/model"
	"hornetSecurity/internal/repository"
	"hornetSecurity/internal/service"
)

type DocumentService interface {
	CreateDocument(document model.Document) (model.Document, error)
	GetDocument(id int) (model.Document, error)
	DeleteDocument(id int) error
}

type DocumentHandler struct {
	service DocumentService
	logger  *slog.Logger
}

func NewDocumentHandler(service DocumentService, logger *slog.Logger) *DocumentHandler {
	return &DocumentHandler{service: service, logger: logger}
}

func (h *DocumentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /documents", h.handle(h.createDocument))
	mux.HandleFunc("GET /documents/{id}", h.handle(h.getDocument))
	mux.HandleFunc("DELETE /documents/{id}", h.handle(h.deleteDocument))
}

type createDocumentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type documentResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type apiError struct {
	status  int
	message string
}

func toDocumentResponse(document model.Document) documentResponse {
	return documentResponse{
		ID:          document.ID,
		Name:        document.Name,
		Description: document.Description,
	}
}

func (h *DocumentHandler) createDocument(w http.ResponseWriter, r *http.Request) error {
	var req createDocumentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return err
	}

	created, err := h.service.CreateDocument(model.Document{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return err
	}

	w.Header().Set("Location", "/documents/"+strconv.Itoa(created.ID))
	return writeJSON(w, http.StatusCreated, toDocumentResponse(created))
}

func (h *DocumentHandler) getDocument(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}

	document, err := h.service.GetDocument(id)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, toDocumentResponse(document))
}

func (h *DocumentHandler) deleteDocument(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}

	if err := h.service.DeleteDocument(id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (e *apiError) Error() string { return e.message }

func badRequest(message string) error {
	return &apiError{status: http.StatusBadRequest, message: message}
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func (h *DocumentHandler) handle(fn handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			h.writeError(w, r, err)
		}
	}
}

func (h *DocumentHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *apiError
	switch {
	case errors.As(err, &apiErr):
		_ = writeJSON(w, apiErr.status, errorResponse{Error: apiErr.message})
	case errors.Is(err, service.ErrInvalidDocument):
		_ = writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, repository.ErrNotFound):
		_ = writeJSON(w, http.StatusNotFound, errorResponse{Error: "document not found"})
	default:
		h.logger.ErrorContext(r.Context(), "request failed",
			"method", r.Method, "path", r.URL.Path, "error", err)
		_ = writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func pathID(r *http.Request) (int, error) {
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

func writeJSON(w http.ResponseWriter, status int, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
	return nil
}
