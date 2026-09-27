package handler

import "hornetSecurity/internal/model"

type errorResponse struct {
	Error string `json:"error"`
}

type apiError struct {
	status  int
	message string
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

func toDocumentResponse(document model.Document) documentResponse {
	return documentResponse{
		ID:          document.ID,
		Name:        document.Name,
		Description: document.Description,
	}
}
