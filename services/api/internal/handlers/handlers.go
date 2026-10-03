package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
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
	csrf  *auth.CSRFService
}

func NewHandler(store *storage.MemoryStore) *Handler {
	return &Handler{repos: factory.NewMemoryFactory(store), auth: auth.NewJWTService(os.Getenv("JWT_SECRET")), csrf: auth.NewCSRFService(os.Getenv("JWT_SECRET"))}
}

func NewHandlerFactory(repos factory.RepositoryFactory) *Handler {
	return &Handler{repos: repos, auth: auth.NewJWTService(os.Getenv("JWT_SECRET")), csrf: auth.NewCSRFService(os.Getenv("JWT_SECRET"))}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(models.HealthResponse{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339)})
}

func (h *Handler) issueCSRFToken(w http.ResponseWriter) string {
	if h.csrf == nil {
		return ""
	}
	token, err := h.csrf.Generate()
	if err != nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
	return token
}

func (h *Handler) requireCSRF(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		return true
	}

	switch r.URL.Path {
	case "/api/v1/auth/csrf", "/api/v1/auth/login", "/api/v1/auth/logout", "/api/v1/parents/register", "/api/v1/devices/pair":
		return true
	}

	token := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
	if token == "" {
		if cookie, err := r.Cookie("csrf_token"); err == nil {
			token = cookie.Value
		}
	}
	if h.csrf == nil || !h.csrf.Validate(token) {
		http.Error(w, "csrf token required", http.StatusForbidden)
		return false
	}
	return true
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

func (h *Handler) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	token, err := h.csrf.Generate()
	if err != nil {
		http.Error(w, "unable to generate csrf token", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
	_ = json.NewEncoder(w).Encode(map[string]string{"csrf_token": token})
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
	h.issueCSRFToken(w)
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

func (h *Handler) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	parentID, err := h.parentIDFromToken(r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	children := h.repos.ChildRepository().ListChildrenByParent(parentID)
	paired := 0
	pending := 0
	childIDs := make(map[string]string, len(children))
	for _, child := range children {
		childIDs[child.ID] = child.Name
		if strings.TrimSpace(child.DeviceID) != "" {
			paired++
		} else {
			pending++
		}
	}

	monitoring := h.repos.MonitoringRepository()
	alerts := monitoring.ListAlerts()
	searches := monitoring.ListSearches()

	filteredAlerts := make([]models.AlertEvent, 0, len(alerts))
	for _, alert := range alerts {
		if _, ok := childIDs[alert.ChildID]; ok {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	recentActivity := make([]models.DashboardActivity, 0, 10)
	for i := len(searches) - 1; i >= 0 && len(recentActivity) < 10; i-- {
		search := searches[i]
		if _, ok := childIDs[search.ChildID]; !ok {
			continue
		}
		recentActivity = append(recentActivity, models.DashboardActivity{
			ID:        search.ID,
			ChildID:   search.ChildID,
			ChildName: childIDs[search.ChildID],
			Query:     search.Query,
			Engine:    search.Engine,
			Timestamp: search.Timestamp,
		})
	}

	summary := models.DashboardSummary{
		TotalChildren:  len(children),
		PairedDevices:  paired,
		PendingPairing: pending,
		AlertsCount:    len(filteredAlerts),
		RecentActivity: recentActivity,
		Alerts:         filteredAlerts,
	}
	_ = json.NewEncoder(w).Encode(summary)
}

func (h *Handler) handleListParentChildren(w http.ResponseWriter, r *http.Request) {
	parentID, err := h.parentIDFromToken(r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(h.repos.ChildRepository().ListChildrenByParent(parentID))
}

func parsePagination(r *http.Request) (int, int) {
	page := 1
	limit := 20

	if v := strings.TrimSpace(r.URL.Query().Get("page")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			if parsed > 100 {
				parsed = 100
			}
			limit = parsed
		}
	}
	return page, limit
}

func (h *Handler) handleListParentAlerts(w http.ResponseWriter, r *http.Request) {
	parentID, err := h.parentIDFromToken(r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	children := h.repos.ChildRepository().ListChildrenByParent(parentID)
	childIDs := make(map[string]struct{}, len(children))
	for _, child := range children {
		childIDs[child.ID] = struct{}{}
	}
	filtered := make([]models.AlertEvent, 0)
	for _, alert := range h.repos.MonitoringRepository().ListAlerts() {
		if _, exists := childIDs[alert.ChildID]; exists {
			filtered = append(filtered, alert)
		}
	}

	page, limit := parsePagination(r)
	if r.URL.Query().Has("page") || r.URL.Query().Has("limit") {
		total := len(filtered)
		totalPages := 1
		if total > 0 {
			totalPages = (total + limit - 1) / limit
		}
		if page > totalPages && totalPages > 0 {
			page = totalPages
		}
		start := (page - 1) * limit
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}
		_ = json.NewEncoder(w).Encode(models.PaginatedAlertResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
			Items:      filtered[start:end],
		})
		return
	}
	_ = json.NewEncoder(w).Encode(filtered)
}

func (h *Handler) handleListParentSearches(w http.ResponseWriter, r *http.Request) {
	parentID, err := h.parentIDFromToken(r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	children := h.repos.ChildRepository().ListChildrenByParent(parentID)
	childIDs := make(map[string]struct{}, len(children))
	for _, child := range children {
		childIDs[child.ID] = struct{}{}
	}
	filtered := make([]models.SearchEvent, 0)
	for _, event := range h.repos.MonitoringRepository().ListSearches() {
		if _, exists := childIDs[event.ChildID]; exists {
			filtered = append(filtered, event)
		}
	}

	page, limit := parsePagination(r)
	if r.URL.Query().Has("page") || r.URL.Query().Has("limit") {
		total := len(filtered)
		totalPages := 1
		if total > 0 {
			totalPages = (total + limit - 1) / limit
		}
		if page > totalPages && totalPages > 0 {
			page = totalPages
		}
		start := (page - 1) * limit
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}
		_ = json.NewEncoder(w).Encode(models.PaginatedSearchResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
			Items:      filtered[start:end],
		})
		return
	}
	_ = json.NewEncoder(w).Encode(filtered)
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
		DeviceID string `json:"device_id,omitempty"`
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

func (h *Handler) handleCreatePairingInvite(w http.ResponseWriter, r *http.Request) {
	parentID, err := h.parentIDFromToken(r)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	childID := path.Base(strings.TrimSuffix(r.URL.Path, "/pairing-invites"))
	if childID == "" || childID == "." || childID == "/" {
		http.Error(w, "child id is required", http.StatusBadRequest)
		return
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		http.Error(w, "unable to create pairing invite", http.StatusInternalServerError)
		return
	}
	code := base64.RawURLEncoding.EncodeToString(secret)
	hash := sha256.Sum256([]byte(code))
	expiresAt := time.Now().UTC().Add(15 * time.Minute)
	if err := h.repos.PairingRepository().CreatePairingInvite(parentID, childID, hex.EncodeToString(hash[:]), expiresAt); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "child not found" {
			status = http.StatusNotFound
		} else if err.Error() == "child already has a paired device" {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":       code,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleRedeemPairingInvite(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Code     string `json:"code"`
		DeviceID string `json:"device_id"`
		Platform string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.Code) == "" || strings.TrimSpace(payload.DeviceID) == "" {
		http.Error(w, "code and device_id are required", http.StatusBadRequest)
		return
	}
	hash := sha256.Sum256([]byte(strings.TrimSpace(payload.Code)))
	child, err := h.repos.PairingRepository().RedeemPairingInvite(hex.EncodeToString(hash[:]), payload.DeviceID, payload.Platform)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "device_id already registered" || err.Error() == "child already has a paired device" {
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
