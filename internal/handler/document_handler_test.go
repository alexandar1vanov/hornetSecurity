package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hornetSecurity/internal/model"
	"hornetSecurity/internal/repository"
	"hornetSecurity/internal/service"
)

type fakeService struct {
	create func(model.Document) (model.Document, error)
	get    func(int) (model.Document, error)
	del    func(int) error
}

func (f *fakeService) CreateDocument(document model.Document) (model.Document, error) {
	return f.create(document)
}

func (f *fakeService) GetDocument(id int) (model.Document, error) { return f.get(id) }

func (f *fakeService) DeleteDocument(id int) error { return f.del(id) }

func newTestMux(svc service.DocumentService) *http.ServeMux {
	mux := http.NewServeMux()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	NewDocumentHandler(svc, logger).RegisterRoutes(mux)
	return mux
}

func serve(svc service.DocumentService, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	newTestMux(svc).ServeHTTP(rec, req)
	return rec
}

func assertJSONResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantBody string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, wantStatus, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
	if got := strings.TrimSpace(rec.Body.String()); got != wantBody {
		t.Errorf("body = %s, want %s", got, wantBody)
	}
}

func TestCreateDocument(t *testing.T) {
	echoWithID := func(d model.Document) (model.Document, error) {
		d.ID = 1
		return d, nil
	}

	tests := []struct {
		name         string
		body         string
		createFn     func(model.Document) (model.Document, error)
		wantStatus   int
		wantBody     string
		wantLocation string
	}{
		{
			name:         "valid document",
			body:         `{"name":"name","description":"description"}`,
			createFn:     echoWithID,
			wantStatus:   http.StatusCreated,
			wantBody:     `{"id":1,"name":"name","description":"description"}`,
			wantLocation: "/api/v1/documents/1",
		},
		{
			name:         "description is optional",
			body:         `{"name":"name"}`,
			createFn:     echoWithID,
			wantStatus:   http.StatusCreated,
			wantBody:     `{"id":1,"name":"name","description":""}`,
			wantLocation: "/api/v1/documents/1",
		},
		{
			name:       "empty body",
			body:       "",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"invalid JSON body"}`,
		},
		{
			name:       "malformed JSON",
			body:       `{"name":`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"invalid JSON body"}`,
		},
		{
			name:       "wrong field type",
			body:       `{"name":123}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"invalid JSON body"}`,
		},
		{
			name:       "unknown field",
			body:       `{"name":"name","owner":"someone"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"invalid JSON body"}`,
		},
		{
			name:       "multiple JSON objects",
			body:       `{"name":"a"}{"name":"b"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"body must contain a single JSON object"}`,
		},
		{
			name:       "trailing garbage",
			body:       `{"name":"a"} garbage`,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"body must contain a single JSON object"}`,
		},
		{
			name:       "body too large",
			body:       `{"name":"` + strings.Repeat("a", 1<<20) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantBody:   `{"error":"request body too large"}`,
		},
		{
			name: "validation error from service",
			body: `{"name":""}`,
			createFn: func(model.Document) (model.Document, error) {
				return model.Document{}, fmt.Errorf("%w: name is required", service.ErrInvalidDocument)
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"invalid document: name is required"}`,
		},
		{
			name: "unexpected service error",
			body: `{"name":"name"}`,
			createFn: func(model.Document) (model.Document, error) {
				return model.Document{}, errors.New("boom")
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(&fakeService{create: tt.createFn}, http.MethodPost, "/api/v1/documents", tt.body)

			assertJSONResponse(t, rec, tt.wantStatus, tt.wantBody)
			if got := rec.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}
		})
	}
}

func TestCreateDocument_PassesRequestToService(t *testing.T) {
	var received model.Document
	svc := &fakeService{create: func(d model.Document) (model.Document, error) {
		received = d
		d.ID = 1
		return d, nil
	}}

	serve(svc, http.MethodPost, "/api/v1/documents", `{"name":"name","description":"description"}`)

	want := model.Document{Name: "name", Description: "description"}
	if received != want {
		t.Errorf("service received %+v, want %+v", received, want)
	}
}

func TestGetDocument(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		getFn      func(int) (model.Document, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "existing document",
			path: "/api/v1/documents/1",
			getFn: func(id int) (model.Document, error) {
				return model.Document{ID: id, Name: "name", Description: "description"}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"id":1,"name":"name","description":"description"}`,
		},
		{
			name: "missing document",
			path: "/api/v1/documents/7",
			getFn: func(id int) (model.Document, error) {
				return model.Document{}, fmt.Errorf("document %d: %w", id, repository.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"document not found"}`,
		},
		{
			name:       "non-numeric id",
			path:       "/api/v1/documents/abc",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"id must be a positive integer"}`,
		},
		{
			name:       "zero id",
			path:       "/api/v1/documents/0",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"id must be a positive integer"}`,
		},
		{
			name:       "negative id",
			path:       "/api/v1/documents/-1",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"id must be a positive integer"}`,
		},
		{
			name: "unexpected service error",
			path: "/api/v1/documents/1",
			getFn: func(int) (model.Document, error) {
				return model.Document{}, errors.New("boom")
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(&fakeService{get: tt.getFn}, http.MethodGet, tt.path, "")
			assertJSONResponse(t, rec, tt.wantStatus, tt.wantBody)
		})
	}
}

func TestDeleteDocument(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		deleteFn   func(int) error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "missing document",
			path:       "/api/v1/documents/7",
			deleteFn:   func(id int) error { return fmt.Errorf("document %d: %w", id, repository.ErrNotFound) },
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"document not found"}`,
		},
		{
			name:       "invalid id",
			path:       "/api/v1/documents/abc",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"id must be a positive integer"}`,
		},
		{
			name:       "unexpected service error",
			path:       "/api/v1/documents/1",
			deleteFn:   func(int) error { return errors.New("boom") },
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(&fakeService{del: tt.deleteFn}, http.MethodDelete, tt.path, "")
			assertJSONResponse(t, rec, tt.wantStatus, tt.wantBody)
		})
	}
}

func TestDeleteDocument_Success(t *testing.T) {
	var requestedID int
	svc := &fakeService{del: func(id int) error {
		requestedID = id
		return nil
	}}

	rec := serve(svc, http.MethodDelete, "/api/v1/documents/5", "")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
	if requestedID != 5 {
		t.Errorf("service Delete() called with id %d, want 5", requestedID)
	}
}

func TestRoutes_MethodNotAllowed(t *testing.T) {
	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/documents"},
		{method: http.MethodPut, path: "/api/v1/documents"},
		{method: http.MethodPost, path: "/api/v1/documents/1"},
		{method: http.MethodPatch, path: "/api/v1/documents/1"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := serve(&fakeService{}, tt.method, tt.path, "")
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestRoutes_NotFound(t *testing.T) {
	rec := serve(&fakeService{}, http.MethodGet, "/unknown", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
