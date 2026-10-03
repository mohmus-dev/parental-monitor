package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"parental-monitor-cli/services/api/internal/auth"
	"parental-monitor-cli/services/api/internal/storage"
)

func TestParentChildRoutesRequireJWT(t *testing.T) {
	store := storage.NewMemoryStore()
	parent, err := store.CreateParentWithPassword("John", "john@example.com", "Secret123!", "Parent App", "windows", "parent-device-001")
	if err != nil {
		t.Fatalf("expected parent registration to succeed: %v", err)
	}
	_ = parent

	h := NewHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parents/me/children", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without jwt, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "unauthorized") {
		t.Fatalf("expected unauthorized response body, got %s", res.Body.String())
	}
}

func TestExpiredJWTReturnsSessionExpired(t *testing.T) {
	store := storage.NewMemoryStore()
	parent, err := store.CreateParentWithPassword("John", "john@example.com", "Secret123!", "Parent App", "windows", "parent-device-001")
	if err != nil {
		t.Fatalf("expected parent registration to succeed: %v", err)
	}

	service := auth.NewJWTService("test-secret-key")
	token, err := service.Generate(auth.Claims{
		ParentID: parent.ID,
		Email:    parent.Email,
		Role:     "parent",
		StandardClaims: auth.StandardClaims{
			IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
			ExpiresAt: time.Now().Add(-time.Minute).Unix(),
		},
	})
	if err != nil {
		t.Fatalf("expected token generation to succeed: %v", err)
	}

	h := &Handler{repos: NewHandler(store).repos, auth: service}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parents/me/children", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized for expired jwt, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "session_expired") {
		t.Fatalf("expected session_expired response body, got %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "expired") {
		t.Fatalf("expected expired message in response body, got %s", res.Body.String())
	}
}

func TestParentIDComesFromJWTClaims(t *testing.T) {
	store := storage.NewMemoryStore()
	parent, err := store.CreateParentWithPassword("John", "john@example.com", "Secret123!", "Parent App", "windows", "parent-device-001")
	if err != nil {
		t.Fatalf("expected parent registration to succeed: %v", err)
	}
	service := auth.NewJWTService("test-secret-key")
	token, err := service.Generate(auth.Claims{ParentID: parent.ID, Email: parent.Email, Role: "parent"})
	if err != nil {
		t.Fatalf("expected token generation to succeed: %v", err)
	}

	h := &Handler{repos: NewHandler(store).repos, auth: service}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parents/me/children", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected ok when token is valid, got %d: %s", res.Code, res.Body.String())
	}
}

func TestPairingInviteRequiresParentAndCanOnlyBeRedeemedOnce(t *testing.T) {
	store := storage.NewMemoryStore()
	parent, err := store.CreateParentWithPassword("John", "john@example.com", "Secret123!")
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	child, err := store.CreateChild(parent.ID, "Emma", 10, "")
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	service := auth.NewJWTService("test-secret-key")
	token, err := service.Generate(auth.Claims{ParentID: parent.ID, Email: parent.Email, Role: "parent"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	h := &Handler{repos: NewHandler(store).repos, auth: service, csrf: auth.NewCSRFService("test-secret-key")}
	csrfToken, err := h.csrf.Generate()
	if err != nil {
		t.Fatalf("generate csrf token: %v", err)
	}

	inviteReq := httptest.NewRequest(http.MethodPost, "/api/v1/parents/me/children/"+child.ID+"/pairing-invites", nil)
	inviteReq.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	inviteReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	inviteReq.Header.Set("X-CSRF-Token", csrfToken)
	inviteRes := httptest.NewRecorder()
	h.ServeHTTP(inviteRes, inviteReq)
	if inviteRes.Code != http.StatusCreated {
		t.Fatalf("expected invite creation, got %d: %s", inviteRes.Code, inviteRes.Body.String())
	}
	var invite struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(inviteRes.Body.Bytes(), &invite); err != nil || invite.Code == "" {
		t.Fatalf("expected pairing code in response, got %s (%v)", inviteRes.Body.String(), err)
	}

	body, err := json.Marshal(map[string]string{"code": invite.Code, "device_id": "device-001", "platform": "windows"})
	if err != nil {
		t.Fatalf("encode pairing request: %v", err)
	}
	pairReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices/pair", bytes.NewReader(body))
	pairRes := httptest.NewRecorder()
	h.ServeHTTP(pairRes, pairReq)
	if pairRes.Code != http.StatusOK {
		t.Fatalf("expected device pairing, got %d: %s", pairRes.Code, pairRes.Body.String())
	}

	replayReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices/pair", bytes.NewReader(body))
	replayRes := httptest.NewRecorder()
	h.ServeHTTP(replayRes, replayReq)
	if replayRes.Code != http.StatusBadRequest {
		t.Fatalf("expected replay rejection, got %d: %s", replayRes.Code, replayRes.Body.String())
	}
}

func TestLoginDoesNotRequireCSRF(t *testing.T) {
	store := storage.NewMemoryStore()
	if _, err := store.CreateParentWithPassword("John", "john@example.com", "Secret123!", "Parent App", "windows", "parent-device-001"); err != nil {
		t.Fatalf("create parent: %v", err)
	}

	h := NewHandler(store)
	body := bytes.NewReader([]byte(`{"email":"john@example.com","password":"Secret123!"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected login to work without CSRF token, got %d: %s", res.Code, res.Body.String())
	}
}

func TestAuthenticatedStateChangingRoutesRequireCSRFToken(t *testing.T) {
	store := storage.NewMemoryStore()
	parent, err := store.CreateParentWithPassword("John", "john@example.com", "Secret123!", "Parent App", "windows", "parent-device-001")
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	service := auth.NewJWTService("test-secret-key")
	token, err := service.Generate(auth.Claims{ParentID: parent.ID, Email: parent.Email, Role: "parent"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	h := &Handler{repos: NewHandler(store).repos, auth: service, csrf: auth.NewCSRFService("test-secret-key")}
	body := bytes.NewReader([]byte(`{"name":"Emma","age":12,"app_name":"Parental Monitor CLI","platform":"windows"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parents/me/children", body)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden without CSRF token for authenticated mutation, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "csrf") {
		t.Fatalf("expected CSRF error message, got %s", res.Body.String())
	}
}
