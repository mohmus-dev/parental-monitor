package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type syncConfig struct {
	Enabled     bool
	Interval    time.Duration
	SQLitePath  string
	PostgresDSN string
}

func main() {
	cfg := loadConfig()
	if !cfg.Enabled {
		log.Println("monitor sync worker disabled")
		return
	}

	log.Printf("monitor sync worker started with interval=%s sqlite=%s", cfg.Interval, cfg.SQLitePath)
	for {
		if err := syncOnce(cfg); err != nil {
			log.Printf("monitor sync failed: %v", err)
		}
		time.Sleep(cfg.Interval)
	}
}

func loadConfig() syncConfig {
	interval := time.Duration(parseInt(os.Getenv("WORKER_SYNC_INTERVAL_SECONDS"), 60)) * time.Second
	cfg := syncConfig{
		Enabled:     strings.EqualFold(strings.TrimSpace(os.Getenv("WORKER_ENABLED")), "true") || strings.EqualFold(strings.TrimSpace(os.Getenv("SYNC_ENABLED")), "true"),
		Interval:    interval,
		SQLitePath:  strings.TrimSpace(os.Getenv("MONITOR_DB_PATH")),
		PostgresDSN: strings.TrimSpace(os.Getenv("POSTGRES_DSN")),
	}
	if cfg.SQLitePath == "" {
		cfg.SQLitePath = "monitor.db"
	}
	if cfg.PostgresDSN == "" {
		cfg.Enabled = false
	}
	return cfg
}

func parseInt(value string, fallback int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func syncOnce(cfg syncConfig) error {
	if !cfg.Enabled {
		return nil
	}

	pg, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer pg.Close()
	if err := pg.Ping(); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	if _, err := pg.Exec(`
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
		);`); err != nil {
		return fmt.Errorf("prepare postgres tables: %w", err)
	}

	sqliteDB, err := sql.Open("sqlite3", cfg.SQLitePath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer sqliteDB.Close()

	if err := syncSearchRows(sqliteDB, pg); err != nil {
		return err
	}
	if err := syncAlertRows(sqliteDB, pg); err != nil {
		return err
	}
	return nil
}

func syncSearchRows(sqliteDB *sql.DB, pg *sql.DB) error {
	rows, err := sqliteDB.Query(`SELECT id, child_id, device_id, query, engine, timestamp, device_name, incognito, window_title FROM searches ORDER BY id`)
	if err != nil {
		return fmt.Errorf("read sqlite searches: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id          int64
			childID     sql.NullString
			deviceID    sql.NullString
			query       string
			engine      sql.NullString
			timestamp   time.Time
			deviceName  sql.NullString
			incognito   bool
			windowTitle sql.NullString
		)
		if err := rows.Scan(&id, &childID, &deviceID, &query, &engine, &timestamp, &deviceName, &incognito, &windowTitle); err != nil {
			return fmt.Errorf("scan sqlite search: %w", err)
		}
		_, err = pg.Exec(`
			INSERT INTO monitor_searches (id, child_id, device_id, query, engine, incognito, window_title, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO NOTHING`,
			strconv.FormatInt(id, 10),
			childID.String,
			deviceID.String,
			query,
			engine.String,
			incognito,
			windowTitle.String,
			timestamp,
		)
		if err != nil {
			return fmt.Errorf("insert search into postgres: %w", err)
		}
	}
	return rows.Err()
}

func syncAlertRows(sqliteDB *sql.DB, pg *sql.DB) error {
	rows, err := sqliteDB.Query(`SELECT id, child_id, device_id, keyword, query, timestamp FROM alerts ORDER BY id`)
	if err != nil {
		return fmt.Errorf("read sqlite alerts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id        int64
			childID   sql.NullString
			deviceID  sql.NullString
			keyword   string
			query     string
			timestamp time.Time
		)
		if err := rows.Scan(&id, &childID, &deviceID, &keyword, &query, &timestamp); err != nil {
			return fmt.Errorf("scan sqlite alert: %w", err)
		}
		_, err = pg.Exec(`
			INSERT INTO monitor_alerts (id, child_id, device_id, keyword, query, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO NOTHING`,
			strconv.FormatInt(id, 10),
			childID.String,
			deviceID.String,
			keyword,
			query,
			timestamp,
		)
		if err != nil {
			return fmt.Errorf("insert alert into postgres: %w", err)
		}
	}
	return rows.Err()
}
