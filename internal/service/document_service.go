package service

import (
	"errors"
	"fmt"
	"hornetSecurity/internal/model"
	"hornetSecurity/internal/repository"
	"strings"
)

var ErrInvalidDocument = errors.New("invalid document")

type DocumentService struct {
	repo repository.DocumentRepository
}

func NewDocumentService(repo repository.DocumentRepository) *DocumentService {
	return &DocumentService{repo: repo}
}

func (s *DocumentService) CreateDocument(document model.Document) (model.Document, error) {
	if strings.TrimSpace(document.Name) == "" {
		return model.Document{}, fmt.Errorf("%w: name is required", ErrInvalidDocument)
	}
	return s.repo.Create(document)
}

func (s *DocumentService) GetDocument(id int) (model.Document, error) { return s.repo.Get(id) }

func (s *DocumentService) DeleteDocument(id int) error { return s.repo.Delete(id) }
