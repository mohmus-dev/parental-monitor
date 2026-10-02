package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	logger "parental-monitor-cli/internal/log"
	"parental-monitor-cli/internal/storage"
)

func TestSearchHandlerStoresIncognitoSearches(t *testing.T) {
	store, err := storage.NewStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory storage: %v", err)
	}
	defer store.Close()

	handler := newSearchHandler(Config{DeviceName: "test-device"}, store, nil, logger.NewLogger("info"))
	request := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"wine party","engine":"Google","incognito":true,"window_title":"Google - wine party"}`))
	request.Header.Set("Origin", "chrome-extension://abcdefghijklmnopabcdefghijklmnop")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}

	searches, err := store.GetRecentSearches(10)
	if err != nil {
		t.Fatalf("failed to read stored searches: %v", err)
	}
	if len(searches) != 1 {
		t.Fatalf("expected one stored search, got %d", len(searches))
	}
	if searches[0].Query != "wine party" || !searches[0].Incognito || searches[0].WindowTitle != "Google - wine party" {
		t.Fatalf("stored search mismatch: %#v", searches[0])
	}
}
