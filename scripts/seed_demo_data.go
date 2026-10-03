package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/option"

	_ "github.com/lib/pq"
)

func main() {
	if err := loadEnvFiles(); err != nil {
		log.Printf("env load: %v", err)
	}

	if err := seedPostgres(); err != nil {
		log.Fatalf("postgres seed failed: %v", err)
	}

	if err := seedFirebase(); err != nil {
		log.Fatalf("firebase seed failed: %v", err)
	}

	log.Println("seed data inserted successfully")
}

func loadEnvFiles() error {
	paths := []string{".env", filepath.Join("services", "api", ".env")}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			if err := godotenv.Load(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedPostgres() error {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return fmt.Errorf("POSTGRES_DSN is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS monitor_searches (
			id TEXT PRIMARY KEY,
			child_id TEXT,
			device_id TEXT,
			query TEXT NOT NULL,
			engine TEXT,
			incognito BOOLEAN DEFAULT false,
			window_title TEXT,
			timestamp TIMESTAMPTZ NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS monitor_alerts (
			id TEXT PRIMARY KEY,
			child_id TEXT,
			device_id TEXT,
			category TEXT,
			keyword TEXT NOT NULL,
			query TEXT NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL
		);`,
		`INSERT INTO monitor_searches (id, child_id, device_id, query, engine, incognito, window_title, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO monitor_alerts (id, child_id, device_id, category, keyword, query, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING`,
	}

	if _, err := db.Exec(queries[0]); err != nil {
		return fmt.Errorf("create search table: %w", err)
	}
	if _, err := db.Exec(queries[1]); err != nil {
		return fmt.Errorf("create alert table: %w", err)
	}

	now := time.Now().UTC()
	searchTemplates := []string{
		"how to make money online fast",
		"best fast food recipes",
		"watch free movies",
		"study app for teens",
		"video game cheat guide",
		"what is social media policy",
		"best homework help website",
		"how to fix slow laptop",
		"free music download site",
		"game strategy online",
		"online casino bonus codes",
		"how to hide browser history",
		"cheap gaming laptop review",
		"best coding tutorial for beginners",
		"movie streaming website free",
	}
	alertTemplates := []struct {
		category string
		keyword  string
		query    string
	}{
		{category: "adult_content", keyword: "porn", query: "porn videos free"},
		{category: "gambling", keyword: "casino", query: "online casino bonus codes"},
		{category: "drugs", keyword: "cocaine", query: "buy cocaine online"},
		{category: "nicotine", keyword: "vape", query: "buy vape pens online"},
		{category: "violence", keyword: "knife", query: "knife fighting tutorial"},
	}

	for i := 0; i < 220; i++ {
		childID := "demo-child-001"
		deviceID := "demo-device-001"
		if i%2 == 1 {
			childID = "demo-child-002"
			deviceID = "demo-device-002"
		}
		query := searchTemplates[i%len(searchTemplates)] + " " + fmt.Sprintf("-%d", i)
		engine := []string{"google", "bing", "duckduckgo", "youtube"}[i%4]
		if _, err := db.Exec(queries[2], fmt.Sprintf("seed-search-%03d", i), childID, deviceID, query, engine, i%3 == 0, "Demo Search", now.Add(-time.Duration(i)*time.Minute)); err != nil {
			return fmt.Errorf("insert generated search row %d: %w", i, err)
		}
	}

	for i := 0; i < 120; i++ {
		childID := "demo-child-001"
		deviceID := "demo-device-001"
		if i%3 == 0 {
			childID = "demo-child-002"
			deviceID = "demo-device-002"
		}
		alert := alertTemplates[i%len(alertTemplates)]
		if _, err := db.Exec(queries[3], fmt.Sprintf("seed-alert-%03d", i), childID, deviceID, alert.category, alert.keyword, alert.query+fmt.Sprintf(" %d", i), now.Add(-time.Duration(i)*time.Minute)); err != nil {
			return fmt.Errorf("insert generated alert row %d: %w", i, err)
		}
	}

	return nil
}

func seedFirebase() error {
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	credentialsPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	if projectID == "" || credentialsPath == "" {
		return fmt.Errorf("FIREBASE_PROJECT_ID and FIREBASE_CREDENTIALS_PATH are required")
	}

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID}, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return fmt.Errorf("firebase init: %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return fmt.Errorf("firestore client: %w", err)
	}
	defer client.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte("Demo@123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("bcrypt parent password: %w", err)
	}

	parentID := "demo-parent-001"
	if _, err := client.Collection("parents").Doc(parentID).Set(ctx, map[string]interface{}{
		"id":            parentID,
		"name":          "Demo Parent",
		"email":         "demo.parent@example.com",
		"password_hash": string(hash),
		"app_name":      "Parent Dashboard",
		"platform":      "windows",
		"device_id":     "demo-parent-device-001",
		"created_at":    time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("seed parent: %w", err)
	}

	childID := "demo-child-001"
	if _, err := client.Collection("children").Doc(childID).Set(ctx, map[string]interface{}{
		"id":         childID,
		"parent_id":  parentID,
		"name":       "Emma",
		"age":        15,
		"device_id":  "demo-device-001",
		"app_name":   "Parental Monitor CLI",
		"platform":   "windows",
		"created_at": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("seed child: %w", err)
	}

	if _, err := client.Collection("children").Doc("demo-child-002").Set(ctx, map[string]interface{}{
		"id":         "demo-child-002",
		"parent_id":  parentID,
		"name":       "Noah",
		"age":        12,
		"device_id":  "demo-device-002",
		"app_name":   "Parental Monitor CLI",
		"platform":   "windows",
		"created_at": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("seed second child: %w", err)
	}

	for _, device := range []struct {
		id      string
		childID string
		name    string
	}{
		{id: "demo-device-001", childID: childID, name: "Emma Laptop"},
		{id: "demo-device-002", childID: "demo-child-002", name: "Noah Tablet"},
	} {
		if _, err := client.Collection("devices").Doc(device.id).Set(ctx, map[string]interface{}{
			"id":         device.id,
			"child_id":   device.childID,
			"name":       device.name,
			"type":       "desktop",
			"platform":   "windows",
			"created_at": time.Now().UTC(),
			"last_seen":  time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("seed device %s: %w", device.id, err)
		}
	}

	searchTemplates := []string{
		"best study apps for teens",
		"gaming laptop review",
		"online tutoring video lessons",
		"free movie download website",
		"how to edit school project",
		"college scholarship search",
		"best music streaming apps",
		"online casino bonus codes",
		"buy vape pens online",
		"watch cartoons free",
	}
	for i := 0; i < 180; i++ {
		child := childID
		device := "demo-device-001"
		if i%2 == 1 {
			child = "demo-child-002"
			device = "demo-device-002"
		}
		searchText := searchTemplates[i%len(searchTemplates)] + fmt.Sprintf(" %d", i)
		if _, err := client.Collection("searches").Doc(fmt.Sprintf("seed-search-fb-%03d", i)).Set(ctx, map[string]interface{}{
			"id":           fmt.Sprintf("seed-search-fb-%03d", i),
			"child_id":     child,
			"device_id":    device,
			"query":        searchText,
			"engine":       []string{"google", "bing", "duckduckgo", "youtube"}[i%4],
			"incognito":    false,
			"window_title": "Demo Search",
			"timestamp":    time.Now().UTC().Add(-time.Duration(i) * time.Minute),
		}); err != nil {
			return fmt.Errorf("seed firebase search %d: %w", i, err)
		}
	}

	alertTemplates := []struct {
		category string
		keyword  string
		query    string
	}{
		{category: "adult_content", keyword: "porn", query: "porn videos free"},
		{category: "gambling", keyword: "casino", query: "online casino bonus codes"},
		{category: "drugs", keyword: "cocaine", query: "buy cocaine online"},
		{category: "nicotine", keyword: "vape", query: "buy vape pens online"},
	}
	for i := 0; i < 80; i++ {
		child := childID
		device := "demo-device-001"
		if i%3 == 0 {
			child = "demo-child-002"
			device = "demo-device-002"
		}
		alert := alertTemplates[i%len(alertTemplates)]
		if _, err := client.Collection("alerts").Doc(fmt.Sprintf("seed-alert-fb-%03d", i)).Set(ctx, map[string]interface{}{
			"id":        fmt.Sprintf("seed-alert-fb-%03d", i),
			"child_id":  child,
			"device_id": device,
			"category":  alert.category,
			"keyword":   alert.keyword,
			"query":     alert.query + fmt.Sprintf(" %d", i),
			"timestamp": time.Now().UTC().Add(-time.Duration(i) * time.Minute),
		}); err != nil {
			return fmt.Errorf("seed firebase alert %d: %w", i, err)
		}
	}

	_, err = client.Collection("parents").Doc(parentID).Get(ctx)
	if err != nil {
		return fmt.Errorf("verify parent exists: %w", err)
	}
	return nil
}

var _ = firestore.Client{}
