package repository

import (
	"errors"
	"fmt"
	"hornetSecurity/internal/model"
	"sync"
)

var ErrNotFound = errors.New("document not found")

type DocumentRepository interface {
	Create(document model.Document) (model.Document, error)
	Get(id int) (model.Document, error)
	Delete(id int) error
}

type inMemoryDocumentRepository struct {
	mu        sync.RWMutex
	documents map[int]model.Document
	nextId    int
}

func NewInMemoryDocumentRepository() DocumentRepository {
	return &inMemoryDocumentRepository{
		documents: make(map[int]model.Document),
	}
}

func (repository *inMemoryDocumentRepository) Create(document model.Document) (model.Document, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	repository.nextId++
	document.ID = repository.nextId
	repository.documents[document.ID] = document
	return document, nil
}

func (repository *inMemoryDocumentRepository) Get(id int) (model.Document, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	doc, ok := repository.documents[id]
	if !ok {
		return model.Document{}, fmt.Errorf("document %d: %w", id, ErrNotFound)
	}
	return doc, nil
}

func (repository *inMemoryDocumentRepository) Delete(id int) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	if _, ok := repository.documents[id]; !ok {
		return fmt.Errorf("document %d: %w", id, ErrNotFound)
	}
	delete(repository.documents, id)
	return nil
}
