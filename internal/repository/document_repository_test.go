package repository

import (
	"errors"
	"sync"
	"testing"

	"hornetSecurity/internal/model"
)

func TestCreate_AssignsSequentialIDs(t *testing.T) {
	repo := NewInMemoryDocumentRepository()

	for want := 1; want <= 3; want++ {
		created, err := repo.Create(model.Document{Name: "doc"})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if created.ID != want {
			t.Errorf("Create() ID = %d, want %d", created.ID, want)
		}
	}
}

func TestCreate_IgnoresProvidedID(t *testing.T) {
	repo := NewInMemoryDocumentRepository()

	created, err := repo.Create(model.Document{ID: 42, Name: "doc"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != 1 {
		t.Errorf("Create() ID = %d, want 1", created.ID)
	}
}

func TestGet(t *testing.T) {
	repo := NewInMemoryDocumentRepository()
	created, err := repo.Create(model.Document{Name: "name", Description: "description"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("existing document", func(t *testing.T) {
		got, err := repo.Get(created.ID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got != created {
			t.Errorf("Get() = %+v, want %+v", got, created)
		}
	})

	t.Run("missing document", func(t *testing.T) {
		_, err := repo.Get(999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Get() error = %v, want %v", err, ErrNotFound)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("existing document", func(t *testing.T) {
		repo := NewInMemoryDocumentRepository()
		created, err := repo.Create(model.Document{Name: "doc"})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if err := repo.Delete(created.ID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if _, err := repo.Get(created.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get() after Delete() error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("missing document", func(t *testing.T) {
		repo := NewInMemoryDocumentRepository()
		if err := repo.Delete(999); !errors.Is(err, ErrNotFound) {
			t.Errorf("Delete() error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("twice", func(t *testing.T) {
		repo := NewInMemoryDocumentRepository()
		created, _ := repo.Create(model.Document{Name: "doc"})
		_ = repo.Delete(created.ID)

		if err := repo.Delete(created.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("second Delete() error = %v, want %v", err, ErrNotFound)
		}
	})
}

func TestDelete_DoesNotReuseIDs(t *testing.T) {
	repo := NewInMemoryDocumentRepository()
	first, _ := repo.Create(model.Document{Name: "first"})
	_ = repo.Delete(first.ID)

	second, err := repo.Create(model.Document{Name: "second"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if second.ID == first.ID {
		t.Errorf("Create() reused deleted ID %d", first.ID)
	}
}

func TestCreate_ConcurrentCallsProduceUniqueIDs(t *testing.T) {
	const workers = 100
	repo := NewInMemoryDocumentRepository()

	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		ids = make(map[int]struct{}, workers)
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, err := repo.Create(model.Document{Name: "doc"})
			if err != nil {
				t.Errorf("Create() error = %v", err)
				return
			}
			mu.Lock()
			ids[created.ID] = struct{}{}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(ids) != workers {
		t.Errorf("got %d unique IDs, want %d", len(ids), workers)
	}
}
