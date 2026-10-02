package factory

import (
	"context"
	"parental-monitor-cli/services/api/internal/models"
	"parental-monitor-cli/services/api/internal/storage"
)

// ParentRepository defines the persistence contract for parent records.
type ParentRepository interface {
	CreateParentWithPassword(name, email, password string, metadata ...string) (models.Parent, error)
	GetParentByEmail(email string) (models.Parent, bool)
	GetParentByID(id string) (models.Parent, bool)
	DeleteParent(id string) error
	ListParents() []models.Parent
	ValidateParentCredentials(email, password string) (models.Parent, bool)
}

// ChildRepository defines the persistence contract for child records.
type ChildRepository interface {
	CreateChild(parentID, name string, age int, deviceID string, metadata ...string) (models.Child, error)
	ListChildren() []models.Child
	ListChildrenByParent(parentID string) []models.Child
	DeleteChildren(parentID string, childIDs []string) error
}

// RepositoryFactory creates repository implementations using a chosen storage backend.
type RepositoryFactory interface {
	ParentRepository() ParentRepository
	ChildRepository() ChildRepository
}

// MemoryFactory keeps the current in-memory store but exposes the repository interfaces.
type MemoryFactory struct {
	parentRepo ParentRepository
	childRepo  ChildRepository
}

func NewMemoryFactory(store *storage.MemoryStore) *MemoryFactory {
	return &MemoryFactory{
		parentRepo: store,
		childRepo:  store,
	}
}

func (f *MemoryFactory) ParentRepository() ParentRepository {
	return f.parentRepo
}

func (f *MemoryFactory) ChildRepository() ChildRepository {
	return f.childRepo
}

// FirebaseFactory is backed by Firestore and implements the same repository interfaces.
type FirebaseFactory struct {
	parentRepo ParentRepository
	childRepo  ChildRepository
}

func NewFirebaseFactory(ctx context.Context, projectID, credentialsPath string) (*FirebaseFactory, error) {
	store, err := storage.NewFirebaseStore(ctx, projectID, credentialsPath)
	if err != nil {
		return nil, err
	}
	return &FirebaseFactory{parentRepo: store, childRepo: store}, nil
}

func (f *FirebaseFactory) ParentRepository() ParentRepository {
	return f.parentRepo
}

func (f *FirebaseFactory) ChildRepository() ChildRepository {
	return f.childRepo
}
