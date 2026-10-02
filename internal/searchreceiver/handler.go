package searchreceiver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"parental-monitor-cli/internal/monitor"
)

type SearchRequest struct {
	Query       string `json:"query"`
	Engine      string `json:"engine"`
	Incognito   bool   `json:"incognito"`
	WindowTitle string `json:"window_title"`
}

type Handler struct {
	deviceName string
	analyzer   *monitor.Analyzer
	onSearch   func(monitor.SearchEvent)
	onAlert    func(monitor.AlertEvent)
}

func NewHandler(
	deviceName string,
	analyzer *monitor.Analyzer,
	onSearch func(monitor.SearchEvent),
	onAlert func(monitor.AlertEvent),
) *Handler {
	return &Handler{
		deviceName: deviceName,
		analyzer:   analyzer,
		onSearch:   onSearch,
		onAlert:    onAlert,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Parental Monitor search receiver is running. Search events are accepted from the Chrome extension.\n"))
		return
	}

	origin := r.Header.Get("Origin")
	if !isChromeExtensionOrigin(origin) {
		http.Error(w, "Chrome extension origin required", http.StatusForbidden)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost || r.URL.Path != "/search" {
		http.NotFound(w, r)
		return
	}
	var request SearchRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Query) == "" || len(request.Query) > 4096 {
		http.Error(w, "query is required and must be at most 4096 bytes", http.StatusBadRequest)
		return
	}

	event := monitor.SearchEvent{
		Query:      request.Query,
		Engine:     request.Engine,
		Timestamp:  time.Now(),
		DeviceName: h.deviceName,
		Incognito:  request.Incognito,
	}
	if strings.TrimSpace(request.WindowTitle) != "" {
		event.WindowTitle = strings.TrimSpace(request.WindowTitle)
	}
	if h.onSearch != nil {
		h.onSearch(event)
	}

	if h.analyzer != nil && h.onAlert != nil {
		for _, match := range h.analyzer.FindMatches(request.Query) {
			h.onAlert(monitor.AlertEvent{
				Category:  match.Category,
				Keyword:   match.Keyword,
				Query:     request.Query,
				Timestamp: event.Timestamp,
			})
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func isChromeExtensionOrigin(origin string) bool {
	const prefix = "chrome-extension://"
	if !strings.HasPrefix(origin, prefix) || len(origin) != len(prefix)+32 {
		return false
	}
	for _, char := range origin[len(prefix):] {
		if char < 'a' || char > 'p' {
			return false
		}
	}
	return true
}
