package service

import (
	"errors"
	"testing"

	"hornetSecurity/internal/model"
	"hornetSecurity/internal/repository"
)

type fakeRepo struct {
	create func(model.Document) (model.Document, error)
	get    func(int) (model.Document, error)
	del    func(int) error

	createCalled bool
}

func (f *fakeRepo) Create(document model.Document) (model.Document, error) {
	f.createCalled = true
	return f.create(document)
}

func (f *fakeRepo) Get(id int) (model.Document, error) { return f.get(id) }

func (f *fakeRepo) Delete(id int) error { return f.del(id) }

func TestCreateDocument_Valid(t *testing.T) {
	var received model.Document
	repo := &fakeRepo{create: func(d model.Document) (model.Document, error) {
		received = d
		d.ID = 1
		return d, nil
	}}
	svc := NewDocumentService(repo)

	input := model.Document{Name: "name", Description: "description"}
	got, err := svc.CreateDocument(input)
	if err != nil {
		t.Fatalf("CreateDocument() error = %v", err)
	}
	if received != input {
		t.Errorf("repo received %+v, want %+v", received, input)
	}
	want := model.Document{ID: 1, Name: "name", Description: "description"}
	if got != want {
		t.Errorf("CreateDocument() = %+v, want %+v", got, want)
	}
}

func TestCreateDocument_InvalidName(t *testing.T) {
	tests := []struct {
		name    string
		docName string
	}{
		{name: "empty", docName: ""},
		{name: "spaces", docName: "   "},
		{name: "tabs and newlines", docName: "\t\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			svc := NewDocumentService(repo)

			_, err := svc.CreateDocument(model.Document{Name: tt.docName})
			if !errors.Is(err, ErrInvalidDocument) {
				t.Errorf("CreateDocument() error = %v, want %v", err, ErrInvalidDocument)
			}
			if repo.createCalled {
				t.Error("repository Create() was called for an invalid document")
			}
		})
	}
}

func TestCreateDocument_RepositoryError(t *testing.T) {
	repoErr := errors.New("storage failure")
	repo := &fakeRepo{create: func(model.Document) (model.Document, error) {
		return model.Document{}, repoErr
	}}
	svc := NewDocumentService(repo)

	if _, err := svc.CreateDocument(model.Document{Name: "name"}); !errors.Is(err, repoErr) {
		t.Errorf("CreateDocument() error = %v, want %v", err, repoErr)
	}
}

func TestGetDocument(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		want := model.Document{ID: 7, Name: "name"}
		var requestedID int
		repo := &fakeRepo{get: func(id int) (model.Document, error) {
			requestedID = id
			return want, nil
		}}

		got, err := NewDocumentService(repo).GetDocument(7)
		if err != nil {
			t.Fatalf("GetDocument() error = %v", err)
		}
		if requestedID != 7 {
			t.Errorf("repo Get() called with id %d, want 7", requestedID)
		}
		if got != want {
			t.Errorf("GetDocument() = %+v, want %+v", got, want)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &fakeRepo{get: func(int) (model.Document, error) {
			return model.Document{}, repository.ErrNotFound
		}}

		if _, err := NewDocumentService(repo).GetDocument(7); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("GetDocument() error = %v, want %v", err, repository.ErrNotFound)
		}
	})
}

func TestDeleteDocument(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var requestedID int
		repo := &fakeRepo{del: func(id int) error {
			requestedID = id
			return nil
		}}

		if err := NewDocumentService(repo).DeleteDocument(3); err != nil {
			t.Fatalf("DeleteDocument() error = %v", err)
		}
		if requestedID != 3 {
			t.Errorf("repo Delete() called with id %d, want 3", requestedID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &fakeRepo{del: func(int) error { return repository.ErrNotFound }}

		if err := NewDocumentService(repo).DeleteDocument(3); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("DeleteDocument() error = %v, want %v", err, repository.ErrNotFound)
		}
	})
}
