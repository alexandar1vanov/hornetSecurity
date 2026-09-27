package main_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hornetSecurity/internal/handler"
	"hornetSecurity/internal/repository"
	"hornetSecurity/internal/service"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	repo := repository.NewInMemoryDocumentRepository()
	mux := http.NewServeMux()
	handler.NewDocumentHandler(service.NewDocumentService(repo), logger).RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func doRequest(t *testing.T, method, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

type document struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return v
}

func TestDocumentLifecycle(t *testing.T) {
	srv := newTestServer(t)

	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/documents", `{"name":"report","description":"quarterly"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	created := decode[document](t, resp)
	if created.ID == 0 || created.Name != "report" || created.Description != "quarterly" {
		t.Fatalf("created document = %+v", created)
	}
	location := resp.Header.Get("Location")

	resp = doRequest(t, http.MethodGet, srv.URL+location, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := decode[document](t, resp); got != created {
		t.Errorf("get document = %+v, want %+v", got, created)
	}

	resp = doRequest(t, http.MethodDelete, srv.URL+location, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	resp = doRequest(t, http.MethodGet, srv.URL+location, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	resp = doRequest(t, http.MethodDelete, srv.URL+location, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestCreateDocument_ValidationThroughAllLayers(t *testing.T) {
	srv := newTestServer(t)

	resp := doRequest(t, http.MethodPost, srv.URL+"/api/v1/documents", `{"name":"   "}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
	body := decode[map[string]string](t, resp)
	if body["error"] != "invalid document: name is required" {
		t.Errorf("error = %q", body["error"])
	}
}

func TestCreateDocument_AssignsDistinctIDs(t *testing.T) {
	srv := newTestServer(t)

	first := decode[document](t, doRequest(t, http.MethodPost, srv.URL+"/api/v1/documents", `{"name":"a"}`))
	second := decode[document](t, doRequest(t, http.MethodPost, srv.URL+"/api/v1/documents", `{"name":"b"}`))

	if first.ID == second.ID {
		t.Errorf("both documents got ID %d", first.ID)
	}
}
