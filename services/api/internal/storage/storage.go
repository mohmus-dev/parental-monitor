package storage

import (
	"errors"
	"strings"
	"sync"
	"time"

	"parental-monitor-cli/services/api/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type MemoryStore struct {
	mu            sync.RWMutex
	parents       map[string]models.Parent
	parentByEmail map[string]string
	children      map[string]models.Child
	devices       map[string]models.Device
	searches      []models.SearchEvent
	alerts        []models.AlertEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		parents:       map[string]models.Parent{},
		parentByEmail: map[string]string{},
		children:      map[string]models.Child{},
		devices:       map[string]models.Device{},
	}
}

func (s *MemoryStore) EnsureParentExists(parentID string) bool {
	_, ok := s.parents[parentID]
	return ok
}

func (s *MemoryStore) CreateParent(name, email string) (models.Parent, error) {
	return s.CreateParentWithPassword(name, email, "")
}

func isStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false
	for _, ch := range password {
		switch {
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	return hasLower && hasUpper && hasDigit && hasSpecial
}

func (s *MemoryStore) CreateParentWithPassword(name, email, password string, metadata ...string) (models.Parent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	email = strings.TrimSpace(strings.ToLower(email))
	appName := ""
	platform := ""
	deviceID := ""
	if len(metadata) > 0 {
		appName = strings.TrimSpace(metadata[0])
	}
	if len(metadata) > 1 {
		platform = strings.TrimSpace(metadata[1])
	}
	if len(metadata) > 2 {
		deviceID = strings.TrimSpace(metadata[2])
	}
	if name == "" || email == "" {
		return models.Parent{}, errors.New("name and email are required")
	}
	if strings.TrimSpace(password) == "" {
		return models.Parent{}, errors.New("password is required")
	}
	if !isStrongPassword(password) {
		return models.Parent{}, errors.New("password must be at least 8 characters and include uppercase, lowercase, number, and special character")
	}
	if _, exists := s.parentByEmail[email]; exists {
		return models.Parent{}, errors.New("parent already registered with this email")
	}

	parent := models.Parent{ID: uuid.NewString(), Name: name, Email: email, AppName: appName, Platform: platform, DeviceID: deviceID, CreatedAt: time.Now()}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.Parent{}, err
	}
	parent.PasswordHash = string(hash)

	s.parents[parent.ID] = parent
	s.parentByEmail[email] = parent.ID
	return parent, nil
}

func (s *MemoryStore) GetParentByEmail(email string) (models.Parent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	email = strings.TrimSpace(strings.ToLower(email))
	id, ok := s.parentByEmail[email]
	if !ok {
		return models.Parent{}, false
	}
	parent, ok := s.parents[id]
	return parent, ok
}

func (s *MemoryStore) GetParentByID(id string) (models.Parent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	parent, ok := s.parents[id]
	return parent, ok
}

func (s *MemoryStore) DeleteParent(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	parent, ok := s.parents[id]
	if !ok {
		return errors.New("parent not found")
	}
	delete(s.parents, id)
	for email, parentID := range s.parentByEmail {
		if parentID == id {
			delete(s.parentByEmail, email)
			break
		}
	}
	for childID, child := range s.children {
		if child.ParentID == parent.ID {
			delete(s.children, childID)
			if child.DeviceID != "" {
				delete(s.devices, child.DeviceID)
			}
		}
	}
	return nil
}

func (s *MemoryStore) ValidateParentCredentials(email, password string) (models.Parent, bool) {
	parent, ok := s.GetParentByEmail(email)
	if !ok || parent.PasswordHash == "" {
		return models.Parent{}, false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(parent.PasswordHash), []byte(password)); err != nil {
		return models.Parent{}, false
	}
	return parent, true
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

func (s *MemoryStore) CreateChild(parentID, name string, age int, deviceID string, metadata ...string) (models.Child, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	appName := ""
	platform := ""
	if len(metadata) > 0 {
		appName = strings.TrimSpace(metadata[0])
	}
	if len(metadata) > 1 {
		platform = strings.TrimSpace(metadata[1])
	}

	if strings.TrimSpace(parentID) == "" {
		return models.Child{}, errors.New("parent_id is required")
	}
	if !s.EnsureParentExists(parentID) {
		return models.Child{}, errors.New("parent not found")
	}
	if strings.TrimSpace(name) == "" {
		return models.Child{}, errors.New("name is required")
	}
	if strings.TrimSpace(deviceID) == "" {
		return models.Child{}, errors.New("device_id is required")
	}
	if _, exists := s.devices[deviceID]; exists {
		return models.Child{}, errors.New("device_id already registered")
	}

	appName = strings.TrimSpace(appName)
	platform = strings.TrimSpace(platform)
	if platform == "" {
		platform = "unknown"
	}
	if appName == "" {
		appName = "Child App"
	}

	c := models.Child{ID: uuid.NewString(), ParentID: parentID, Name: name, Age: age, DeviceID: deviceID, AppName: appName, Platform: platform, CreatedAt: time.Now()}
	s.children[c.ID] = c
	s.devices[deviceID] = models.Device{ID: deviceID, ChildID: c.ID, Name: name, Type: "mobile", Platform: platform, CreatedAt: time.Now(), LastSeen: time.Now()}
	return c, nil
}

func (s *MemoryStore) ListChildrenByParent(parentID string) []models.Child {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]models.Child, 0)
	for _, child := range s.children {
		if child.ParentID == parentID {
			items = append(items, child)
		}
	}
	return items
}

func (s *MemoryStore) DeleteChildren(parentID string, childIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(parentID) == "" {
		return errors.New("parent_id is required")
	}
	if len(childIDs) == 0 {
		return errors.New("child_ids are required")
	}

	allowed := make(map[string]struct{}, len(childIDs))
	for _, childID := range childIDs {
		childID = strings.TrimSpace(childID)
		if childID != "" {
			allowed[childID] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return errors.New("child_ids are required")
	}

	deleted := 0
	for childID, child := range s.children {
		if child.ParentID != parentID {
			continue
		}
		if _, exists := allowed[child.ID]; !exists {
			continue
		}
		delete(s.children, childID)
		if child.DeviceID != "" {
			delete(s.devices, child.DeviceID)
		}
		deleted++
	}
	if deleted == 0 {
		return errors.New("no child matches the provided ids for this parent")
	}
	return nil
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
