package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func loadConfig(envPath string) (Config, error) {
	values, err := godotenv.Read(envPath)
	if err != nil {
		return Config{}, fmt.Errorf("read environment file %q: %w", envPath, err)
	}

	cfg := Config{
		APIURL:     strings.TrimRight(values["API_URL"], "/"),
		ServerURL:  strings.TrimRight(values["SERVER_URL"], "/"),
		APIKey:     values["API_KEY"],
		ChildID:    values["CHILD_ID"],
		DeviceID:   values["DEVICE_ID"],
		DeviceName: values["DEVICE_NAME"],
		LogLevel:   values["LOG_LEVEL"],
	}
	if cfg.DeviceName == "" {
		cfg.DeviceName = "Child device"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	if cfg.MonitorIntervalMs, err = envInt(values, "MONITOR_INTERVAL_MS", 50); err != nil {
		return Config{}, err
	}
	if cfg.BatchSize, err = envInt(values, "BATCH_SIZE", 10); err != nil {
		return Config{}, err
	}
	if cfg.FlushIntervalSec, err = envInt(values, "FLUSH_INTERVAL_SEC", 5); err != nil {
		return Config{}, err
	}
	if cfg.ScanKeywords, err = envStringList(values, "SCAN_KEYWORDS"); err != nil {
		return Config{}, err
	}
	if cfg.CustomKeywords, err = envStringList(values, "CUSTOM_KEYWORDS"); err != nil {
		return Config{}, err
	}
	if cfg.ScanKeywordGroups, err = envStringGroups(values, "SCAN_KEYWORD_GROUPS"); err != nil {
		return Config{}, err
	}
	if cfg.CustomKeywordGroups, err = envStringGroups(values, "CUSTOM_KEYWORD_GROUPS"); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func envInt(values map[string]string, key string, fallback int) (int, error) {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func envStringList(values map[string]string, key string) ([]string, error) {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return nil, nil
	}
	var items []string
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil, fmt.Errorf("%s must be a JSON string array: %w", key, err)
	}
	return items, nil
}

func envStringGroups(values map[string]string, key string) (map[string][]string, error) {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return nil, nil
	}
	var groups map[string][]string
	if err := json.Unmarshal([]byte(value), &groups); err != nil {
		return nil, fmt.Errorf("%s must be a JSON object of string arrays: %w", key, err)
	}
	return groups, nil
}
