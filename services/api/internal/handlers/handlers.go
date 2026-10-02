package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"parental-monitor-cli/services/api/internal/models"
	"parental-monitor-cli/services/api/internal/storage"
)

type Handler struct {
	store *storage.MemoryStore
}

func NewHandler(store *storage.MemoryStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/health":
		h.handleHealth(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/parents":
		h.handleListParents(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/parents":
		h.handleCreateParent(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/children":
		h.handleListChildren(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/children":
		h.handleCreateChild(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/devices":
		h.handleListDevices(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/devices":
		h.handleCreateDevice(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/searches":
		h.handleListSearches(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/searches":
		h.handleCreateSearch(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/alerts":
		h.handleListAlerts(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/alerts":
		h.handleCreateAlert(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(models.HealthResponse{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339)})
}

func (h *Handler) handleListParents(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(h.store.ListParents())
}

func (h *Handler) handleCreateParent(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	parent := h.store.CreateParent(payload.Name, payload.Email)
	_ = json.NewEncoder(w).Encode(parent)
}

func (h *Handler) handleListChildren(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(h.store.ListChildren())
}

func (h *Handler) handleCreateChild(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ParentID string `json:"parent_id"`
		Name     string `json:"name"`
		Age      int    `json:"age"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	child := h.store.CreateChild(payload.ParentID, payload.Name, payload.Age)
	_ = json.NewEncoder(w).Encode(child)
}

func (h *Handler) handleListDevices(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(h.store.ListDevices())
}

func (h *Handler) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ChildID  string `json:"child_id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Platform string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	device := h.store.CreateDevice(payload.ChildID, payload.Name, payload.Type, payload.Platform)
	_ = json.NewEncoder(w).Encode(device)
}

func (h *Handler) handleListSearches(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(h.store.ListSearches())
}

func (h *Handler) handleCreateSearch(w http.ResponseWriter, r *http.Request) {
	var item models.SearchEvent
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	stored := h.store.SaveSearch(item)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(stored)
}

func (h *Handler) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(h.store.ListAlerts())
}

func (h *Handler) handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	var item models.AlertEvent
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	stored := h.store.SaveAlert(item)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(stored)
}
