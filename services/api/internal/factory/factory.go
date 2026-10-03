package factory

import (
	"context"
	"time"

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

type PairingRepository interface {
	CreatePairingInvite(parentID, childID, tokenHash string, expiresAt time.Time) error
	RedeemPairingInvite(tokenHash, deviceID, platform string) (models.Child, error)
}

type MonitoringRepository interface {
	SaveSearch(event models.SearchEvent) models.SearchEvent
	ListSearches() []models.SearchEvent
	SaveAlert(event models.AlertEvent) models.AlertEvent
	ListAlerts() []models.AlertEvent
}

// RepositoryFactory creates repository implementations using a chosen storage backend.
type RepositoryFactory interface {
	ParentRepository() ParentRepository
	ChildRepository() ChildRepository
	PairingRepository() PairingRepository
	MonitoringRepository() MonitoringRepository
}

// MemoryFactory keeps the current in-memory store but exposes the repository interfaces.
type MemoryFactory struct {
	parentRepo     ParentRepository
	childRepo      ChildRepository
	pairingRepo    PairingRepository
	monitoringRepo MonitoringRepository
}

func NewMemoryFactory(store *storage.MemoryStore) *MemoryFactory {
	return &MemoryFactory{
		parentRepo:     store,
		childRepo:      store,
		pairingRepo:    store,
		monitoringRepo: store,
	}
}

func (f *MemoryFactory) ParentRepository() ParentRepository {
	return f.parentRepo
}

func (f *MemoryFactory) ChildRepository() ChildRepository {
	return f.childRepo
}

func (f *MemoryFactory) PairingRepository() PairingRepository {
	return f.pairingRepo
}

func (f *MemoryFactory) MonitoringRepository() MonitoringRepository {
	return f.monitoringRepo
}

// FirebaseFactory is backed by Firestore and implements the same repository interfaces.
type FirebaseFactory struct {
	parentRepo     ParentRepository
	childRepo      ChildRepository
	pairingRepo    PairingRepository
	monitoringRepo MonitoringRepository
}

func NewFirebaseFactory(ctx context.Context, projectID, credentialsPath string) (*FirebaseFactory, error) {
	store, err := storage.NewFirebaseStore(ctx, projectID, credentialsPath)
	if err != nil {
		return nil, err
	}
	return &FirebaseFactory{parentRepo: store, childRepo: store, pairingRepo: store, monitoringRepo: store}, nil
}

func (f *FirebaseFactory) ParentRepository() ParentRepository {
	return f.parentRepo
}

func (f *FirebaseFactory) ChildRepository() ChildRepository {
	return f.childRepo
}

func (f *FirebaseFactory) PairingRepository() PairingRepository {
	return f.pairingRepo
}

func (f *FirebaseFactory) MonitoringRepository() MonitoringRepository {
	return f.monitoringRepo
}

// HybridFactory allows identity/pairing to live in Firebase while monitoring data can be stored in Postgres.
type HybridFactory struct {
	parentRepo     ParentRepository
	childRepo      ChildRepository
	pairingRepo    PairingRepository
	monitoringRepo MonitoringRepository
}

func NewHybridFactory(parentRepo ParentRepository, childRepo ChildRepository, pairingRepo PairingRepository, monitoringRepo MonitoringRepository) *HybridFactory {
	return &HybridFactory{parentRepo: parentRepo, childRepo: childRepo, pairingRepo: pairingRepo, monitoringRepo: monitoringRepo}
}

func (f *HybridFactory) ParentRepository() ParentRepository {
	return f.parentRepo
}

func (f *HybridFactory) ChildRepository() ChildRepository {
	return f.childRepo
}

func (f *HybridFactory) PairingRepository() PairingRepository {
	return f.pairingRepo
}

func (f *HybridFactory) MonitoringRepository() MonitoringRepository {
	return f.monitoringRepo
}
