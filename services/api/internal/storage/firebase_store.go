package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"parental-monitor-cli/services/api/internal/models"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FirebaseStore struct {
	client    *firestore.Client
	parentCol string
	childCol  string
	deviceCol string
	inviteCol string
}

func NewFirebaseStore(ctx context.Context, projectID, credentialsPath string) (*FirebaseStore, error) {
	if projectID == "" {
		projectID = os.Getenv("FIREBASE_PROJECT_ID")
	}
	if credentialsPath == "" {
		credentialsPath = os.Getenv("FIREBASE_CREDENTIALS_PATH")
	}
	credentialsPath = resolveCredentialsPath(credentialsPath)

	cfg := &firebase.Config{ProjectID: projectID}
	var opts []option.ClientOption
	if credentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
	}

	app, err := firebase.NewApp(ctx, cfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("firebase init failed: %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("firestore client init failed: %w", err)
	}

	return &FirebaseStore{
		client:    client,
		parentCol: "parents",
		childCol:  "children",
		deviceCol: "devices",
		inviteCol: "pairing_invites",
	}, nil
}

func (s *FirebaseStore) CreateParentWithPassword(name, email, password string, metadata ...string) (models.Parent, error) {
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
	if _, exists := s.GetParentByEmail(email); exists {
		return models.Parent{}, errors.New("parent already registered with this email")
	}

	parent := models.Parent{
		ID:        uuid.NewString(),
		Name:      name,
		Email:     email,
		AppName:   appName,
		Platform:  platform,
		DeviceID:  deviceID,
		CreatedAt: time.Now().UTC(),
	}
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return models.Parent{}, err
		}
		parent.PasswordHash = string(hash)
	}

	_, err := s.client.Collection(s.parentCol).Doc(parent.ID).Set(context.Background(), map[string]interface{}{
		"id":            parent.ID,
		"name":          parent.Name,
		"email":         parent.Email,
		"password_hash": parent.PasswordHash,
		"app_name":      parent.AppName,
		"platform":      parent.Platform,
		"device_id":     parent.DeviceID,
		"created_at":    parent.CreatedAt,
	})
	if err != nil {
		return models.Parent{}, err
	}
	return parent, nil
}

func (s *FirebaseStore) GetParentByEmail(email string) (models.Parent, bool) {
	email = strings.TrimSpace(strings.ToLower(email))
	iter := s.client.Collection(s.parentCol).Where("email", "==", email).Limit(1).Documents(context.Background())
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			return models.Parent{}, false
		}
		if err != nil {
			return models.Parent{}, false
		}
		parent, err := parentFromDoc(doc)
		if err != nil {
			return models.Parent{}, false
		}
		return parent, true
	}
}

func (s *FirebaseStore) GetParentByID(id string) (models.Parent, bool) {
	doc, err := s.client.Collection(s.parentCol).Doc(id).Get(context.Background())
	if err != nil {
		return models.Parent{}, false
	}
	parent, err := parentFromDoc(doc)
	if err != nil {
		return models.Parent{}, false
	}
	return parent, true
}

func (s *FirebaseStore) DeleteParent(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("parent_id is required")
	}
	if _, ok := s.GetParentByID(id); !ok {
		return errors.New("parent not found")
	}

	iter := s.client.Collection(s.childCol).Where("parent_id", "==", id).Documents(context.Background())
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		if _, err := s.client.Collection(s.childCol).Doc(doc.Ref.ID).Delete(context.Background()); err != nil {
			return err
		}
	}

	_, err := s.client.Collection(s.parentCol).Doc(id).Delete(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func (s *FirebaseStore) ListParents() []models.Parent {
	iter := s.client.Collection(s.parentCol).Documents(context.Background())
	defer iter.Stop()

	parents := make([]models.Parent, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			continue
		}
		parent, err := parentFromDoc(doc)
		if err != nil {
			continue
		}
		parents = append(parents, parent)
	}
	return parents
}

func (s *FirebaseStore) ValidateParentCredentials(email, password string) (models.Parent, bool) {
	parent, ok := s.GetParentByEmail(email)
	if !ok || parent.PasswordHash == "" {
		return models.Parent{}, false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(parent.PasswordHash), []byte(password)); err != nil {
		return models.Parent{}, false
	}
	return parent, true
}

func (s *FirebaseStore) CreateChild(parentID, name string, age int, deviceID string, metadata ...string) (models.Child, error) {
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
	if _, exists := s.GetParentByID(parentID); !exists {
		return models.Child{}, errors.New("parent not found")
	}
	if strings.TrimSpace(name) == "" {
		return models.Child{}, errors.New("name is required")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID != "" {
		iter := s.client.Collection(s.childCol).Where("device_id", "==", deviceID).Limit(1).Documents(context.Background())
		defer iter.Stop()
		if _, err := iter.Next(); err == nil {
			return models.Child{}, errors.New("device_id already registered")
		} else if err != iterator.Done && err != nil {
			return models.Child{}, err
		}
	}

	appName = strings.TrimSpace(appName)
	platform = strings.TrimSpace(platform)
	if appName == "" {
		appName = "Child App"
	}
	if platform == "" {
		platform = "unknown"
	}

	child := models.Child{
		ID:        uuid.NewString(),
		ParentID:  parentID,
		Name:      name,
		Age:       age,
		DeviceID:  deviceID,
		AppName:   appName,
		Platform:  platform,
		CreatedAt: time.Now().UTC(),
	}
	if deviceID != "" {
		deviceRef := s.client.Collection(s.deviceCol).Doc(deviceDocumentID(deviceID))
		_, err := deviceRef.Create(context.Background(), map[string]interface{}{
			"child_id":   child.ID,
			"created_at": child.CreatedAt,
		})
		if status.Code(err) == codes.AlreadyExists {
			return models.Child{}, errors.New("device_id already registered")
		}
		if err != nil {
			return models.Child{}, err
		}
	}
	_, err := s.client.Collection(s.childCol).Doc(child.ID).Set(context.Background(), map[string]interface{}{
		"id":         child.ID,
		"parent_id":  child.ParentID,
		"name":       child.Name,
		"age":        child.Age,
		"device_id":  child.DeviceID,
		"app_name":   child.AppName,
		"platform":   child.Platform,
		"created_at": child.CreatedAt,
	})
	if err != nil {
		if deviceID != "" {
			if _, cleanupErr := s.client.Collection(s.deviceCol).Doc(deviceDocumentID(deviceID)).Delete(context.Background()); cleanupErr != nil {
				return models.Child{}, fmt.Errorf("create child failed: %v; release device reservation: %w", err, cleanupErr)
			}
		}
		return models.Child{}, err
	}
	return child, nil
}

func (s *FirebaseStore) CreatePairingInvite(parentID, childID, tokenHash string, expiresAt time.Time) error {
	childDoc, err := s.client.Collection(s.childCol).Doc(childID).Get(context.Background())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return errors.New("child not found")
		}
		return err
	}
	child, err := childFromDoc(childDoc)
	if err != nil {
		return err
	}
	if child.ParentID != parentID {
		return errors.New("child not found")
	}
	if child.DeviceID != "" {
		return errors.New("child already has a paired device")
	}
	if tokenHash == "" || !expiresAt.After(time.Now()) {
		return errors.New("invalid pairing invite")
	}

	_, err = s.client.Collection(s.inviteCol).Doc(tokenHash).Set(context.Background(), map[string]interface{}{
		"parent_id":  parentID,
		"child_id":   childID,
		"expires_at": expiresAt.UTC(),
		"created_at": time.Now().UTC(),
	})
	return err
}

func (s *FirebaseStore) RedeemPairingInvite(tokenHash, deviceID, platform string) (models.Child, error) {
	deviceID = strings.TrimSpace(deviceID)
	if tokenHash == "" || deviceID == "" {
		return models.Child{}, errors.New("invite_code and device_id are required")
	}
	iter := s.client.Collection(s.childCol).Where("device_id", "==", deviceID).Limit(1).Documents(context.Background())
	defer iter.Stop()
	if _, err := iter.Next(); err == nil {
		return models.Child{}, errors.New("device_id already registered")
	} else if err != iterator.Done {
		return models.Child{}, err
	}

	var paired models.Child
	err := s.client.RunTransaction(context.Background(), func(ctx context.Context, tx *firestore.Transaction) error {
		inviteRef := s.client.Collection(s.inviteCol).Doc(tokenHash)
		inviteDoc, err := tx.Get(inviteRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return errors.New("pairing invite is invalid or already used")
			}
			return err
		}
		invite := inviteDoc.Data()
		if _, used := invite["used_at"]; used {
			return errors.New("pairing invite is invalid or already used")
		}
		expiresAt, ok := invite["expires_at"].(time.Time)
		if !ok || !expiresAt.After(time.Now()) {
			return errors.New("pairing invite has expired")
		}

		childID := fmt.Sprint(invite["child_id"])
		parentID := fmt.Sprint(invite["parent_id"])
		childRef := s.client.Collection(s.childCol).Doc(childID)
		childDoc, err := tx.Get(childRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return errors.New("child not found")
			}
			return err
		}
		child, err := childFromDoc(childDoc)
		if err != nil {
			return err
		}
		if child.ParentID != parentID {
			return errors.New("child not found")
		}
		if child.DeviceID != "" {
			return errors.New("child already has a paired device")
		}

		platform = strings.TrimSpace(platform)
		if platform == "" {
			platform = "unknown"
		}
		child.DeviceID = deviceID
		child.Platform = platform
		child.AppName = "Parental Monitor CLI"
		paired = child
		now := time.Now().UTC()
		if err := tx.Set(childRef, map[string]interface{}{
			"id":         child.ID,
			"parent_id":  child.ParentID,
			"name":       child.Name,
			"age":        child.Age,
			"device_id":  child.DeviceID,
			"app_name":   child.AppName,
			"platform":   child.Platform,
			"created_at": child.CreatedAt,
		}); err != nil {
			return err
		}
		deviceRef := s.client.Collection(s.deviceCol).Doc(deviceDocumentID(deviceID))
		if err := tx.Create(deviceRef, map[string]interface{}{
			"child_id":   child.ID,
			"created_at": now,
		}); err != nil {
			return err
		}
		return tx.Update(inviteRef, []firestore.Update{{Path: "used_at", Value: now}})
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return models.Child{}, errors.New("device_id already registered")
		}
		return models.Child{}, err
	}
	return paired, nil
}

func (s *FirebaseStore) ListChildren() []models.Child {
	iter := s.client.Collection(s.childCol).Documents(context.Background())
	defer iter.Stop()

	children := make([]models.Child, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			continue
		}
		child, err := childFromDoc(doc)
		if err != nil {
			continue
		}
		children = append(children, child)
	}
	return children
}

func (s *FirebaseStore) ListChildrenByParent(parentID string) []models.Child {
	iter := s.client.Collection(s.childCol).Where("parent_id", "==", parentID).Documents(context.Background())
	defer iter.Stop()

	children := make([]models.Child, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			continue
		}
		child, err := childFromDoc(doc)
		if err != nil {
			continue
		}
		children = append(children, child)
	}
	return children
}

func (s *FirebaseStore) DeleteChildren(parentID string, childIDs []string) error {
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
	iter := s.client.Collection(s.childCol).Where("parent_id", "==", parentID).Documents(context.Background())
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		if _, exists := allowed[doc.Ref.ID]; !exists {
			continue
		}
		if _, err := s.client.Collection(s.childCol).Doc(doc.Ref.ID).Delete(context.Background()); err != nil {
			return err
		}
		deviceID := strings.TrimSpace(fmt.Sprint(doc.Data()["device_id"]))
		if deviceID != "" && deviceID != "<nil>" {
			if _, err := s.client.Collection(s.deviceCol).Doc(deviceDocumentID(deviceID)).Delete(context.Background()); err != nil {
				return err
			}
		}
		deleted++
	}
	if deleted == 0 {
		return errors.New("no child matches the provided ids for this parent")
	}
	return nil
}

func resolveCredentialsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}

	candidates := []string{path}
	if filepath.IsAbs(path) {
		candidates = append(candidates, strings.TrimLeft(path, "/"), strings.TrimPrefix(path, "/"))
	}
	if !filepath.IsAbs(path) {
		candidates = append(candidates, filepath.Join(".", path))
	}

	current := os.Getenv("PWD")
	if current == "" {
		current, _ = os.Getwd()
	}
	for dir := current; dir != ""; dir = filepath.Dir(dir) {
		for _, candidate := range candidates {
			full := filepath.Join(dir, candidate)
			if _, err := os.Stat(full); err == nil {
				return full
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		current = parent
		if filepath.Base(current) == "services" {
			current = parent
		}
		dir = current
	}

	for _, candidate := range candidates {
		if candidate != path {
			return candidate
		}
	}
	return path
}

func parentFromDoc(doc *firestore.DocumentSnapshot) (models.Parent, error) {
	if doc == nil {
		return models.Parent{}, errors.New("missing parent document")
	}
	data := doc.Data()
	parent := models.Parent{
		ID:           doc.Ref.ID,
		Name:         strings.TrimSpace(fmt.Sprint(data["name"])),
		Email:        strings.TrimSpace(strings.ToLower(fmt.Sprint(data["email"]))),
		PasswordHash: fmt.Sprint(data["password_hash"]),
	}
	if createdAt, ok := data["created_at"]; ok {
		if ts, ok := createdAt.(time.Time); ok {
			parent.CreatedAt = ts
		}
	}
	if parent.ID == "" || parent.Email == "" {
		return models.Parent{}, errors.New("invalid parent document")
	}
	return parent, nil
}

func childFromDoc(doc *firestore.DocumentSnapshot) (models.Child, error) {
	if doc == nil {
		return models.Child{}, errors.New("missing child document")
	}
	data := doc.Data()
	child := models.Child{
		ID:       doc.Ref.ID,
		ParentID: fmt.Sprint(data["parent_id"]),
		Name:     strings.TrimSpace(fmt.Sprint(data["name"])),
		DeviceID: fmt.Sprint(data["device_id"]),
		AppName:  strings.TrimSpace(fmt.Sprint(data["app_name"])),
		Platform: strings.TrimSpace(fmt.Sprint(data["platform"])),
	}
	if age, ok := data["age"]; ok {
		if v, ok := age.(int64); ok {
			child.Age = int(v)
		}
	}
	if createdAt, ok := data["created_at"]; ok {
		if ts, ok := createdAt.(time.Time); ok {
			child.CreatedAt = ts
		}
	}
	if child.ID == "" || child.ParentID == "" {
		return models.Child{}, errors.New("invalid child document")
	}
	return child, nil
}

func (s *FirebaseStore) SaveSearch(event models.SearchEvent) models.SearchEvent {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	_, err := s.client.Collection("searches").Doc(event.ID).Set(context.Background(), map[string]interface{}{
		"id":           event.ID,
		"child_id":     event.ChildID,
		"device_id":    event.DeviceID,
		"query":        event.Query,
		"engine":       event.Engine,
		"incognito":    event.Incognito,
		"window_title": event.WindowTitle,
		"timestamp":    event.Timestamp,
	})
	if err != nil {
		return event
	}
	return event
}

func (s *FirebaseStore) SaveAlert(event models.AlertEvent) models.AlertEvent {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	_, err := s.client.Collection("alerts").Doc(event.ID).Set(context.Background(), map[string]interface{}{
		"id":        event.ID,
		"child_id":  event.ChildID,
		"device_id": event.DeviceID,
		"category":  event.Category,
		"keyword":   event.Keyword,
		"query":     event.Query,
		"timestamp": event.Timestamp,
	})
	if err != nil {
		return event
	}
	return event
}

func (s *FirebaseStore) ListSearches() []models.SearchEvent {
	iter := s.client.Collection("searches").Documents(context.Background())
	defer iter.Stop()

	items := make([]models.SearchEvent, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			continue
		}
		var item models.SearchEvent
		if err := doc.DataTo(&item); err == nil {
			item.ID = doc.Ref.ID
			items = append(items, item)
		}
	}
	return items
}

func (s *FirebaseStore) ListAlerts() []models.AlertEvent {
	iter := s.client.Collection("alerts").Documents(context.Background())
	defer iter.Stop()

	items := make([]models.AlertEvent, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			continue
		}
		var item models.AlertEvent
		if err := doc.DataTo(&item); err == nil {
			item.ID = doc.Ref.ID
			items = append(items, item)
		}
	}
	return items
}

func deviceDocumentID(deviceID string) string {
	sum := sha256.Sum256([]byte(deviceID))
	return hex.EncodeToString(sum[:])
}
