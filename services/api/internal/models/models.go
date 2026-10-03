package models

import "time"

type Parent struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	AppName      string    `json:"app_name,omitempty"`
	Platform     string    `json:"platform,omitempty"`
	DeviceID     string    `json:"device_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Child struct {
	ID        string    `json:"id"`
	ParentID  string    `json:"parent_id"`
	Name      string    `json:"name"`
	Age       int       `json:"age,omitempty"`
	DeviceID  string    `json:"device_id,omitempty"`
	AppName   string    `json:"app_name,omitempty"`
	Platform  string    `json:"platform,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type PairingInvite struct {
	ID        string
	ParentID  string
	ChildID   string
	TokenHash string
	ExpiresAt time.Time
}

type Device struct {
	ID        string    `json:"id"`
	ChildID   string    `json:"child_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
}

type SearchEvent struct {
	ID          string    `json:"id"`
	ChildID     string    `json:"child_id"`
	DeviceID    string    `json:"device_id"`
	Query       string    `json:"query"`
	Engine      string    `json:"engine"`
	Incognito   bool      `json:"incognito"`
	WindowTitle string    `json:"window_title,omitempty"`
	Category    string    `json:"category,omitempty"`
	Keyword     string    `json:"keyword,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

type AlertEvent struct {
	ID        string    `json:"id"`
	ChildID   string    `json:"child_id"`
	DeviceID  string    `json:"device_id"`
	Category  string    `json:"category"`
	Keyword   string    `json:"keyword"`
	Query     string    `json:"query"`
	Timestamp time.Time `json:"timestamp"`
}

type HealthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

type DashboardActivity struct {
	ID        string    `json:"id"`
	ChildID   string    `json:"child_id"`
	ChildName string    `json:"child_name"`
	Query     string    `json:"query"`
	Engine    string    `json:"engine"`
	Timestamp time.Time `json:"timestamp"`
}

type DashboardSummary struct {
	TotalChildren  int                 `json:"total_children"`
	PairedDevices  int                 `json:"paired_devices"`
	PendingPairing int                 `json:"pending_pairing"`
	AlertsCount    int                 `json:"alerts_count"`
	RecentActivity []DashboardActivity `json:"recent_activity"`
	Alerts         []AlertEvent        `json:"alerts"`
}

type PaginatedAlertResponse struct {
	Page       int          `json:"page"`
	Limit      int          `json:"limit"`
	Total      int          `json:"total"`
	TotalPages int          `json:"total_pages"`
	Items      []AlertEvent `json:"items"`
}

type PaginatedSearchResponse struct {
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	Total      int           `json:"total"`
	TotalPages int           `json:"total_pages"`
	Items      []SearchEvent `json:"items"`
}
