package storage

import (
    "database/sql"
    "encoding/json"
    "os"
    "time"

    _ "github.com/mattn/go-sqlite3"
    "parental-monitor-cli/internal/monitor"
)

type Storage struct {
    db *sql.DB
}

type SearchRow struct {
    ID          int64     `json:"id"`
    Query       string    `json:"query"`
    Engine      string    `json:"engine"`
    Timestamp   time.Time `json:"timestamp"`
    DeviceName  string    `json:"device_name"`
    Incognito   bool      `json:"incognito"`
    WindowTitle string    `json:"window_title"`
}

func NewStorage(dbPath string) (*Storage, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }

    // Create tables
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS searches (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            query TEXT NOT NULL,
            engine TEXT,
            timestamp DATETIME NOT NULL,
            device_name TEXT,
            incognito BOOLEAN DEFAULT 0,
            window_title TEXT
        );
        CREATE TABLE IF NOT EXISTS alerts (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            keyword TEXT NOT NULL,
            query TEXT NOT NULL,
            timestamp DATETIME NOT NULL
        );
    `)
    if err != nil {
        return nil, err
    }

    return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
    return s.db.Close()
}

func (s *Storage) SaveSearch(event monitor.SearchEvent) error {
    _, err := s.db.Exec(
        "INSERT INTO searches (query, engine, timestamp, device_name, incognito, window_title) VALUES (?, ?, ?, ?, ?, ?)",
        event.Query, event.Engine, event.Timestamp, event.DeviceName, event.Incognito, event.WindowTitle,
    )
    return err
}

func (s *Storage) SaveAlert(event monitor.AlertEvent) error {
    _, err := s.db.Exec(
        "INSERT INTO alerts (keyword, query, timestamp) VALUES (?, ?, ?)",
        event.Keyword, event.Query, event.Timestamp,
    )
    return err
}

func (s *Storage) GetRecentSearches(limit int) ([]SearchRow, error) {
    rows, err := s.db.Query("SELECT * FROM searches ORDER BY timestamp DESC LIMIT ?", limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var searches []SearchRow
    for rows.Next() {
        var s SearchRow
        rows.Scan(&s.ID, &s.Query, &s.Engine, &s.Timestamp, &s.DeviceName, &s.Incognito, &s.WindowTitle)
        searches = append(searches, s)
    }
    return searches, nil
}

func (s *Storage) ExportJSON(filename string) error {
    searches, err := s.GetRecentSearches(1000)
    if err != nil {
        return err
    }
    data, _ := json.MarshalIndent(searches, "", "  ")
    return os.WriteFile(filename, data, 0644)
}