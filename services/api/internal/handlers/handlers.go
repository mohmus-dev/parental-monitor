package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"parental-monitor-cli/services/api/internal/auth"
	"parental-monitor-cli/services/api/internal/factory"
	"parental-monitor-cli/services/api/internal/models"
	"parental-monitor-cli/services/api/internal/storage"
)

type Handler struct {
	repos factory.RepositoryFactory
	auth  *auth.JWTService
}

func NewHandler(store *storage.MemoryStore) *Handler {
	return &Handler{repos: factory.NewMemoryFactory(store), auth: auth.NewJWTService(os.Getenv("JWT_SECRET"))}
}

func NewHandlerFactory(repos factory.RepositoryFactory) *Handler {
	return &Handler{repos: repos, auth: auth.NewJWTService(os.Getenv("JWT_SECRET"))}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(models.HealthResponse{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339)})
}

func (h *Handler) clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})
}

func (h *Handler) writeAuthError(w http.ResponseWriter, err error) {
	lower := strings.ToLower(err.Error())
	h.clearAuthCookie(w)
	w.WriteHeader(http.StatusUnauthorized)

	if strings.Contains(lower, "expired") {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "session_expired",
			"message": "Your session has expired. Please log in again.",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "unauthorized",
		"message": "Authentication required or token is invalid.",
	})
}

func (h *Handler) parentIDFromToken(r *http.Request) (string, error) {
	if h.auth == nil {
		return "", errors.New("authentication service unavailable")
	}
	cookie, err := r.Cookie("auth_token")
	if err != nil {
		fmt.Println("Error retrieving auth_token cookie:", err)
		return "", errors.New("missing auth_token cookie")
	}
	if strings.TrimSpace(cookie.Value) == "" {
		return "", errors.New("missing auth_token cookie")
	}
	claims, err := h.auth.Validate(cookie.Value)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "expired") {
			return "", errors.New("jwt expired")
		}
		return "", errors.New("invalid token")
	}
	if strings.TrimSpace(claims.ParentID) == "" {
		return "", errors.New("token missing parent id")
	}
	return claims.ParentID, nil
}

func (h *Handler) handleListParents(w http.ResponseWriter, r *http.Request) {
	if _, err := h.parentIDFromToken(r); err != nil {
		h.writeAuthError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(h.repos.ParentRepository().ListParents())
}

func (h *Handler) handleCurrentParent(w http.ResponseWriter, r *http.Request) {
	parentID, err := h.parentIDFromToken(r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	parent, ok := h.repos.ParentRepository().GetParentByID(parentID)
	if !ok {
		http.Error(w, "parent not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(parent)
}

func (h *Handler) handleDeleteParent(w http.ResponseWriter, r *http.Request) {
	parentID, err := h.parentIDFromToken(r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	if parentID == "" {
		http.Error(w, "parent id is required", http.StatusBadRequest)
		return
	}

	if err := h.repos.ParentRepository().DeleteParent(parentID); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "parent not found" {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: false})
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "parent and associated children deleted successfully"})
}

func (h *Handler) handleCreateParent(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password,omitempty"`
		AppName  string `json:"app_name,omitempty"`
		Platform string `json:"platform,omitempty"`
		DeviceID string `json:"device_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.Password) == "" {
		http.Error(w, "password is required", http.StatusBadRequest)
		return
	}

	parent, err := h.repos.ParentRepository().CreateParentWithPassword(payload.Name, payload.Email, payload.Password, payload.AppName, payload.Platform, payload.DeviceID)
	if err != nil {
		status := http.StatusConflict
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	response := struct {
		Parent  models.Parent `json:"parent"`
		Message string        `json:"message"`
	}{
		Parent:  parent,
		Message: "parent registered successfully",
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleParentLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	parent, ok := h.repos.ParentRepository().ValidateParentCredentials(payload.Email, payload.Password)
	if !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.auth.Generate(auth.Claims{
		ParentID: parent.ID,
		Email:    parent.Email,
		Role:     "parent",
	})
	if err != nil {
		http.Error(w, "unable to generate token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
	response := struct {
		Token  string `json:"token"`
		Expiry string `json:"expiry"`
	}{
		Token:  token,
		Expiry: time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339),
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleListChildren(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(h.repos.ChildRepository().ListChildren())
}

func (h *Handler) handleListParentChildren(w http.ResponseWriter, r *http.Request) {
	parentID, err := h.parentIDFromToken(r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(h.repos.ChildRepository().ListChildrenByParent(parentID))
}

func (h *Handler) handleDeleteParentChildren(w http.ResponseWriter, r *http.Request) {
	parentID, err := h.parentIDFromToken(r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	var payload struct {
		ChildIDs []string `json:"child_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := h.repos.ChildRepository().DeleteChildren(parentID, payload.ChildIDs); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "no child matches the provided ids for this parent" {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "selected child devices deleted successfully"})
}

func (h *Handler) handleCreateChild(w http.ResponseWriter, r *http.Request) {
	parentID, err := h.parentIDFromToken(r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	var payload struct {
		Name     string `json:"name"`
		Age      int    `json:"age"`
		DeviceID string `json:"device_id"`
		AppName  string `json:"app_name"`
		Platform string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	child, err := h.repos.ChildRepository().CreateChild(parentID, payload.Name, payload.Age, payload.DeviceID, payload.AppName, payload.Platform)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "parent not found" || err.Error() == "device_id already registered" {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	_ = json.NewEncoder(w).Encode(child)
}

func (h *Handler) handleListDevices(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode([]models.Device{})
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
	device := models.Device{ID: payload.ChildID, ChildID: payload.ChildID, Name: payload.Name, Type: payload.Type, Platform: payload.Platform, CreatedAt: time.Now(), LastSeen: time.Now()}
	_ = json.NewEncoder(w).Encode(device)
}

func (h *Handler) handleListSearches(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode([]models.SearchEvent{})
}

func (h *Handler) handleCreateSearch(w http.ResponseWriter, r *http.Request) {
	var item models.SearchEvent
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	stored := item
	stored.ID = item.ID
	stored.Timestamp = time.Now()
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(stored)
}

func (h *Handler) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode([]models.AlertEvent{})
}

func (h *Handler) handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	var item models.AlertEvent
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	stored := item
	stored.ID = item.ID
	stored.Timestamp = time.Now()
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(stored)
}
