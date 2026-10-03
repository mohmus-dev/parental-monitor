package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type CSRFService struct {
	secret string
}

func NewCSRFService(secret string) *CSRFService {
	if secret == "" {
		secret = defaultJWTSecret
	}
	return &CSRFService{secret: secret}
}

func (s *CSRFService) Generate() (string, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	signature := hmac.New(sha256.New, []byte(s.secret))
	_, _ = signature.Write(nonce)
	return hex.EncodeToString(nonce) + "." + hex.EncodeToString(signature.Sum(nil)), nil
}

func (s *CSRFService) Validate(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	payload, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	signature := hmac.New(sha256.New, []byte(s.secret))
	_, _ = signature.Write(payload)
	expected := hex.EncodeToString(signature.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(parts[1]))
}
