package storage

import (
	"database/sql"
	"fmt"
	"time"

	"parental-monitor-cli/services/api/internal/models"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type PostgresMonitoringStore struct {
	db *sql.DB
}

func NewPostgresMonitoringStore(dsn string) (*PostgresMonitoringStore, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres dsn is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS monitor_searches (
			id TEXT PRIMARY KEY,
			child_id TEXT,
			device_id TEXT,
			query TEXT NOT NULL,
			engine TEXT,
			incognito BOOLEAN DEFAULT false,
			window_title TEXT,
			timestamp TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE IF NOT EXISTS monitor_alerts (
			id TEXT PRIMARY KEY,
			child_id TEXT,
			device_id TEXT,
			category TEXT,
			keyword TEXT NOT NULL,
			query TEXT NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL
		);
	`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("prepare postgres monitoring tables: %w", err)
	}

	return &PostgresMonitoringStore{db: db}, nil
}

func (s *PostgresMonitoringStore) SaveSearch(event models.SearchEvent) models.SearchEvent {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	_, err := s.db.Exec(
		`INSERT INTO monitor_searches (id, child_id, device_id, query, engine, incognito, window_title, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			child_id = EXCLUDED.child_id,
			device_id = EXCLUDED.device_id,
			query = EXCLUDED.query,
			engine = EXCLUDED.engine,
			incognito = EXCLUDED.incognito,
			window_title = EXCLUDED.window_title,
			timestamp = EXCLUDED.timestamp`,
		event.ID,
		event.ChildID,
		event.DeviceID,
		event.Query,
		event.Engine,
		event.Incognito,
		event.WindowTitle,
		event.Timestamp,
	)
	if err != nil {
		return event
	}
	return event
}

func (s *PostgresMonitoringStore) ListSearches() []models.SearchEvent {
	rows, err := s.db.Query(`SELECT id, child_id, device_id, query, engine, incognito, window_title, timestamp FROM monitor_searches ORDER BY timestamp DESC LIMIT 2000`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	items := make([]models.SearchEvent, 0)
	for rows.Next() {
		var item models.SearchEvent
		if err := rows.Scan(&item.ID, &item.ChildID, &item.DeviceID, &item.Query, &item.Engine, &item.Incognito, &item.WindowTitle, &item.Timestamp); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *PostgresMonitoringStore) SaveAlert(event models.AlertEvent) models.AlertEvent {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	_, err := s.db.Exec(
		`INSERT INTO monitor_alerts (id, child_id, device_id, category, keyword, query, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			child_id = EXCLUDED.child_id,
			device_id = EXCLUDED.device_id,
			category = EXCLUDED.category,
			keyword = EXCLUDED.keyword,
			query = EXCLUDED.query,
			timestamp = EXCLUDED.timestamp`,
		event.ID,
		event.ChildID,
		event.DeviceID,
		event.Category,
		event.Keyword,
		event.Query,
		event.Timestamp,
	)
	if err != nil {
		return event
	}
	return event
}

func (s *PostgresMonitoringStore) ListAlerts() []models.AlertEvent {
	rows, err := s.db.Query(`SELECT id, child_id, device_id, category, keyword, query, timestamp FROM monitor_alerts ORDER BY timestamp DESC LIMIT 2000`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	items := make([]models.AlertEvent, 0)
	for rows.Next() {
		var item models.AlertEvent
		if err := rows.Scan(&item.ID, &item.ChildID, &item.DeviceID, &item.Category, &item.Keyword, &item.Query, &item.Timestamp); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}
