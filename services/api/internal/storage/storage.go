package storage

import (
	"sync"
	"time"

	"parental-monitor-cli/services/api/internal/models"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu       sync.RWMutex
	parents  map[string]models.Parent
	children map[string]models.Child
	devices  map[string]models.Device
	searches []models.SearchEvent
	alerts   []models.AlertEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		parents:  map[string]models.Parent{},
		children: map[string]models.Child{},
		devices:  map[string]models.Device{},
	}
}

func (s *MemoryStore) CreateParent(name, email string) models.Parent {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := models.Parent{ID: uuid.NewString(), Name: name, Email: email, CreatedAt: time.Now()}
	s.parents[p.ID] = p
	return p
}

func (s *MemoryStore) ListParents() []models.Parent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]models.Parent, 0, len(s.parents))
	for _, p := range s.parents {
		items = append(items, p)
	}
	return items
}

func (s *MemoryStore) CreateChild(parentID, name string, age int) models.Child {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := models.Child{ID: uuid.NewString(), ParentID: parentID, Name: name, Age: age, CreatedAt: time.Now()}
	s.children[c.ID] = c
	return c
}

func (s *MemoryStore) ListChildren() []models.Child {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]models.Child, 0, len(s.children))
	for _, c := range s.children {
		items = append(items, c)
	}
	return items
}

func (s *MemoryStore) CreateDevice(childID, name, deviceType, platform string) models.Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := models.Device{ID: uuid.NewString(), ChildID: childID, Name: name, Type: deviceType, Platform: platform, CreatedAt: time.Now(), LastSeen: time.Now()}
	s.devices[d.ID] = d
	return d
}

func (s *MemoryStore) ListDevices() []models.Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]models.Device, 0, len(s.devices))
	for _, d := range s.devices {
		items = append(items, d)
	}
	return items
}

func (s *MemoryStore) SaveSearch(event models.SearchEvent) models.SearchEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	event.ID = uuid.NewString()
	event.Timestamp = time.Now()
	s.searches = append(s.searches, event)
	return event
}

func (s *MemoryStore) ListSearches() []models.SearchEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]models.SearchEvent(nil), s.searches...)
	return items
}

func (s *MemoryStore) SaveAlert(event models.AlertEvent) models.AlertEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	event.ID = uuid.NewString()
	event.Timestamp = time.Now()
	s.alerts = append(s.alerts, event)
	return event
}

func (s *MemoryStore) ListAlerts() []models.AlertEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]models.AlertEvent(nil), s.alerts...)
	return items
}
