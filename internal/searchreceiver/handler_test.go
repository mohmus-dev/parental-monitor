package searchreceiver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"parental-monitor-cli/internal/monitor"
)

func TestHandlerReportsEveryMatchAndPreservesSubmittedQuery(t *testing.T) {
	var searches []monitor.SearchEvent
	var alerts []monitor.AlertEvent
	handler := NewHandler(
		"test-device",
		monitor.NewGroupedAnalyzer(nil, map[string][]string{
			"drugs":    {"drugs", "morphine"},
			"violence": {"beaten"},
		}),
		func(event monitor.SearchEvent) { searches = append(searches, event) },
		func(event monitor.AlertEvent) { alerts = append(alerts, event) },
	)

	request := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"wwe drugs beaten","engine":"Google","incognito":true}`))
	request.Header.Set("Origin", "chrome-extension://abcdefghijklmnopabcdefghijklmnop")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}
	if len(searches) != 1 || searches[0].Query != "wwe drugs beaten" || !searches[0].Incognito {
		t.Fatalf("unexpected saved search event: %#v", searches)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected two alerts, got %#v", alerts)
	}
	for _, alert := range alerts {
		if alert.Query != "wwe drugs beaten" {
			t.Fatalf("alert did not preserve submitted query: %#v", alert)
		}
	}
}

func TestHandlerRejectsNonExtensionOrigin(t *testing.T) {
	handler := NewHandler("test-device", nil, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"morphine"}`))
	request.Header.Set("Origin", "https://example.com")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.Code)
	}
}

func TestHandlerRootShowsStatusWithoutExtensionOrigin(t *testing.T) {
	handler := NewHandler("test-device", nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if !strings.Contains(response.Body.String(), "receiver is running") {
		t.Fatalf("expected receiver status message, got %q", response.Body.String())
	}
}

func TestSearchRequestJSONShape(t *testing.T) {
	payload, err := json.Marshal(SearchRequest{Query: "morphine", Engine: "Google", Incognito: true})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["query"] != "morphine" || decoded["engine"] != "Google" || decoded["incognito"] != true {
		t.Fatalf("unexpected search request JSON: %s", payload)
	}
}
