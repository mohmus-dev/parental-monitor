package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfigReadsSettingsFromEnvFile(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	env := `API_URL=http://localhost:8080
SERVER_URL=https://ingest.example
API_KEY=test-key
DEVICE_NAME=Child device
MONITOR_INTERVAL_MS=75
BATCH_SIZE=20
FLUSH_INTERVAL_SEC=8
SCAN_KEYWORDS='["example phrase"]'
SCAN_KEYWORD_GROUPS='{"safety":["term one","term two"]}'
CUSTOM_KEYWORDS='["custom phrase"]'
CUSTOM_KEYWORD_GROUPS='{"family":["custom term"]}'
CHILD_ID=child-123
DEVICE_ID=device-456
LOG_LEVEL=debug
`
	if err := os.WriteFile(envPath, []byte(env), 0600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	cfg, err := loadConfig(envPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.APIURL != "http://localhost:8080" || cfg.ServerURL != "https://ingest.example" || cfg.APIKey != "test-key" {
		t.Fatalf("unexpected server settings: %#v", cfg)
	}
	if cfg.DeviceName != "Child device" || cfg.ChildID != "child-123" || cfg.DeviceID != "device-456" || cfg.LogLevel != "debug" {
		t.Fatalf("unexpected device settings: %#v", cfg)
	}
	if cfg.MonitorIntervalMs != 75 || cfg.BatchSize != 20 || cfg.FlushIntervalSec != 8 {
		t.Fatalf("unexpected numeric settings: %#v", cfg)
	}
	if !reflect.DeepEqual(cfg.ScanKeywords, []string{"example phrase"}) ||
		!reflect.DeepEqual(cfg.ScanKeywordGroups, map[string][]string{"safety": {"term one", "term two"}}) ||
		!reflect.DeepEqual(cfg.CustomKeywords, []string{"custom phrase"}) ||
		!reflect.DeepEqual(cfg.CustomKeywordGroups, map[string][]string{"family": {"custom term"}}) {
		t.Fatalf("unexpected keyword settings: %#v", cfg)
	}
}

func TestLoadConfigRejectsMalformedEnvSettings(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
	}{
		{name: "integer", value: "BATCH_SIZE=not-a-number\n"},
		{name: "keywords", value: "SCAN_KEYWORDS=not-json\n"},
		{name: "groups", value: "SCAN_KEYWORD_GROUPS='[]'\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			envPath := filepath.Join(t.TempDir(), ".env")
			if err := os.WriteFile(envPath, []byte(test.value), 0600); err != nil {
				t.Fatalf("write env: %v", err)
			}
			if _, err := loadConfig(envPath); err == nil {
				t.Fatal("expected malformed environment setting to fail")
			}
		})
	}
}

func TestExampleEnvPreservesDefaultKeywordSettings(t *testing.T) {
	cfg, err := loadConfig(".env.example")
	if err != nil {
		t.Fatalf("load example env: %v", err)
	}
	if len(cfg.ScanKeywords) == 0 || len(cfg.ScanKeywordGroups) == 0 {
		t.Fatal("example env should retain default keyword configuration")
	}
}
