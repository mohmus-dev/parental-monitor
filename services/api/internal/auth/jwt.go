package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const defaultJWTSecret = "parent-monitor-dev-secret-change-me"

// StandardClaims mirrors the basic JWT v1-style claims used by the app.
type StandardClaims struct {
	ExpiresAt int64  `json:"exp,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	NotBefore int64  `json:"nbf,omitempty"`
	Issuer    string `json:"iss,omitempty"`
}

// Claims contains app-specific JWT data for a parent session.
type Claims struct {
	ParentID string `json:"parent_id,omitempty"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
	StandardClaims
}

type JWTService struct {
	secret string
}

func NewJWTService(secret string) *JWTService {
	if secret == "" {
		secret = defaultJWTSecret
	}
	return &JWTService{secret: secret}
}

func (s *JWTService) Generate(claims Claims) (string, error) {
	now := time.Now().Unix()
	if claims.IssuedAt == 0 {
		claims.IssuedAt = now
	}
	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = now + int64((30*24*time.Hour)/time.Second)
	}
	if claims.NotBefore == 0 {
		claims.NotBefore = now
	}

	headerJSON, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadPart := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signaturePart := signHS256(s.secret, headerPart+"."+payloadPart)
	return headerPart + "." + payloadPart + "." + signaturePart, nil
}

func (s *JWTService) Validate(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid jwt format")
	}

	headerPart, payloadPart, signaturePart := parts[0], parts[1], parts[2]
	expectedSig := signHS256(s.secret, headerPart+"."+payloadPart)
	if !hmac.Equal([]byte(signaturePart), []byte(expectedSig)) {
		return Claims{}, errors.New("invalid jwt signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return Claims{}, err
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return Claims{}, err
	}

	if claims.ExpiresAt != 0 && time.Now().Unix() > claims.ExpiresAt {
		return Claims{}, errors.New("jwt expired")
	}
	return claims, nil
}

func signHS256(secret, message string) string {
	hash := hmac.New(sha256.New, []byte(secret))
	_, _ = hash.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}
