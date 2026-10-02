package auth

import (
	"testing"
	"time"
)

func TestJWTServiceGeneratesAndValidatesToken(t *testing.T) {
	service := NewJWTService("test-secret-key")
	claims := Claims{
		ParentID: "parent-123",
		Email:    "john@example.com",
		Role:     "parent",
		StandardClaims: StandardClaims{
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour).Unix(),
		},
	}

	token, err := service.Generate(claims)
	if err != nil {
		t.Fatalf("expected token generation to succeed: %v", err)
	}
	if token == "" {
		t.Fatalf("expected token to be non-empty")
	}

	parsed, err := service.Validate(token)
	if err != nil {
		t.Fatalf("expected token validation to succeed: %v", err)
	}
	if parsed.ParentID != "parent-123" {
		t.Fatalf("expected parent id to match")
	}
	if parsed.Email != "john@example.com" {
		t.Fatalf("expected email to match")
	}
}

func TestJWTServiceRejectsExpiredToken(t *testing.T) {
	service := NewJWTService("test-secret-key")
	claims := Claims{
		ParentID: "parent-123",
		Email:    "john@example.com",
		Role:     "parent",
		StandardClaims: StandardClaims{
			ExpiresAt: time.Now().Add(-time.Hour).Unix(),
		},
	}

	token, err := service.Generate(claims)
	if err != nil {
		t.Fatalf("expected token generation to succeed: %v", err)
	}

	if _, err := service.Validate(token); err == nil {
		t.Fatalf("expected expired token validation to fail")
	}
}
