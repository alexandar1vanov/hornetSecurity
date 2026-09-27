package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"hornetSecurity/internal/model"
	"hornetSecurity/internal/service"
)

type DocumentHandler interface {
	RegisterRoutes(mux *http.ServeMux)
}
type documentHandler struct {
	service service.DocumentService
	logger  *slog.Logger
}

func NewDocumentHandler(service service.DocumentService, logger *slog.Logger) DocumentHandler {
	return &documentHandler{service: service, logger: logger}
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func (h *documentHandler) handle(fn handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			h.writeError(w, r, err)
		}
	}
}

func (h *documentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/documents", h.handle(h.createDocument))
	mux.HandleFunc("GET /api/v1/documents/{id}", h.handle(h.getDocument))
	mux.HandleFunc("DELETE /api/v1/documents/{id}", h.handle(h.deleteDocument))
}

func (h *documentHandler) createDocument(w http.ResponseWriter, r *http.Request) error {
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

	w.Header().Set("Location", "/api/v1/documents/"+strconv.Itoa(created.ID))
	h.writeJSON(w, r, http.StatusCreated, toDocumentResponse(created))
	return nil
}

func (h *documentHandler) getDocument(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromPath(r)
	if err != nil {
		return err
	}

	document, err := h.service.GetDocument(id)
	if err != nil {
		return err
	}
	h.writeJSON(w, r, http.StatusOK, toDocumentResponse(document))
	return nil
}

func (h *documentHandler) deleteDocument(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromPath(r)
	if err != nil {
		return err
	}

	if err := h.service.DeleteDocument(id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
