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
