package handlers

import (
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
