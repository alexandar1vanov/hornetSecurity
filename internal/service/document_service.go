package service

import (
	"errors"
	"fmt"
	"hornetSecurity/internal/model"
	"hornetSecurity/internal/repository"
	"strings"
)

var ErrInvalidDocument = errors.New("invalid document")

type DocumentService interface {
	CreateDocument(document model.Document) (model.Document, error)
	GetDocument(id int) (model.Document, error)
	DeleteDocument(id int) error
}
type documentService struct {
	repo repository.DocumentRepository
}

func NewDocumentService(repo repository.DocumentRepository) DocumentService {
	return &documentService{repo: repo}
}

func (s *documentService) CreateDocument(document model.Document) (model.Document, error) {
	if strings.TrimSpace(document.Name) == "" {
		return model.Document{}, fmt.Errorf("%w: name is required", ErrInvalidDocument)
	}
	return s.repo.Create(document)
}

func (s *documentService) GetDocument(id int) (model.Document, error) { return s.repo.Get(id) }

func (s *documentService) DeleteDocument(id int) error { return s.repo.Delete(id) }
