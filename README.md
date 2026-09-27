# hornetSecurity – Document Service

A small REST API written in Go for creating, retrieving and deleting **documents**.
It uses only the Go standard library (no external dependencies) and follows a
layered architecture: **handler → service → repository**.

Documents are stored **in memory**, so all data is lost when the server stops.

## Features

- REST endpoints to create, get and delete documents
- JSON request/response bodies with consistent error responses
- Input validation (document name is required, IDs must be positive integers)
- Strict JSON decoding: unknown fields are rejected, request bodies are limited to 1 MB
- Thread-safe in-memory storage (`sync.RWMutex`)
- Structured JSON logging with `log/slog`
- HTTP server timeouts and graceful shutdown on `SIGINT` / `SIGTERM`
- Configurable port via the `PORT` environment variable
- Unit tests for every layer plus end-to-end integration tests
- Multi-stage Dockerfile producing a small, non-root distroless image

## Project structure

```
.
├── main.go                         # Entry point: wiring, HTTP server, graceful shutdown
├── integration_test.go             # End-to-end tests against a real HTTP test server
├── Dockerfile                      # Multi-stage build (golang:alpine → distroless)
├── .dockerignore
├── .gitignore
├── go.mod
└── internal/
    ├── model/
    │   └── document.go             # Document entity
    ├── handler/
    │   ├── document_handler.go     # HTTP routes and request handlers
    │   ├── types.go                # Request/response types
    │   ├── utils.go                # JSON encoding/decoding, error mapping, helpers
    │   └── document_handler_test.go
    ├── service/
    │   ├── document_service.go     # Business logic and validation
    │   └── document_service_test.go
    └── repository/
        ├── document_repository.go  # Storage interface + in-memory implementation
        └── document_repository_test.go
```

### Layers

| Layer | Responsibility |
|-------|----------------|
| **Handler** | Registers routes, parses path params and JSON bodies, maps errors to HTTP status codes |
| **Service** | Validates documents (e.g. `name` must not be blank) and delegates to the repository |
| **Repository** | Stores documents; the in-memory repository assigns auto-incrementing IDs |

Each layer depends on an interface of the layer below, so implementations (e.g. a
database-backed repository) can be swapped without touching the other layers.

## Requirements

- Go **1.27** or newer
- Docker (optional, for running in a container)

## Running the application

### Locally

```bash
go run .
```

The server listens on port `8080` by default. To use another port:

```bash
# bash
PORT=9090 go run .

# PowerShell
$env:PORT = "9090"; go run .
```

### With Docker

```bash
docker build -t hornet-security .
docker run --rm -p 8080:8080 hornet-security
```

## API

Base URL: `http://localhost:8080`

### Document object

```json
{
  "id": 1,
  "name": "Report",
  "description": "Quarterly security report"
}
```

### Create a document

`POST /api/v1/documents`

```bash
curl -i -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{"name": "Report", "description": "Quarterly security report"}'
```

**Response:** `201 Created` with a `Location: /api/v1/documents/{id}` header and the created document in the body.

### Get a document

`GET /api/v1/documents/{id}`

```bash
curl -i http://localhost:8080/api/v1/documents/1
```

**Response:** `200 OK` with the document in the body.

### Delete a document

`DELETE /api/v1/documents/{id}`

```bash
curl -i -X DELETE http://localhost:8080/api/v1/documents/1
```

**Response:** `204 No Content`.

### Errors

All errors are returned as JSON:

```json
{ "error": "document not found" }
```

| Status | When |
|--------|------|
| `400 Bad Request` | Invalid JSON, unknown fields, more than one JSON object, missing/blank `name`, or `id` is not a positive integer |
| `404 Not Found` | No document exists with the given `id` |
| `413 Request Entity Too Large` | Request body is larger than 1 MB |
| `500 Internal Server Error` | Unexpected error (details are logged, not returned) |

## Testing

Run all unit and integration tests:

```bash
go test ./...
```

With the race detector and verbose output:

```bash
go test -race -v ./...
```

- `internal/*/..._test.go` – unit tests for each layer
- `integration_test.go` – spins up the full stack with `httptest.NewServer` and exercises the API over HTTP

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Port the HTTP server listens on |

Server timeouts (set in `main.go`): read header 5s, read 10s, write 10s, idle 60s,
graceful shutdown 10s.
