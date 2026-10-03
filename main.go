package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	logger "parental-monitor-cli/internal/log"
	"parental-monitor-cli/internal/monitor"
	"parental-monitor-cli/internal/network"
	"parental-monitor-cli/internal/searchreceiver"
	"parental-monitor-cli/internal/storage"
)

const searchReceiverAddress = "127.0.0.1:8765"

type Config struct {
	APIURL              string
	ServerURL           string
	APIKey              string
	ChildID             string
	DeviceID            string
	DeviceName          string
	MonitorIntervalMs   int
	BatchSize           int
	FlushIntervalSec    int
	ScanKeywords        []string
	ScanKeywordGroups   map[string][]string
	CustomKeywords      []string
	CustomKeywordGroups map[string][]string
	LogLevel            string
}

func main() {
	// Parse command line flags
	envPath := flag.String("env", ".env", "path to environment file")
	pair := flag.Bool("pair", false, "pair this device using a parent-generated code")
	daemon := flag.Bool("daemon", false, "run as background service")
	stop := flag.Bool("stop", false, "stop the background service")
	flag.Parse()

	if *pair {
		if err := runPairing(*envPath); err != nil {
			log.Fatalf("Device pairing failed: %v", err)
		}
		return
	}

	// ============ DAEMON MODE HANDLING ============
	if *daemon {
		daemonize(*envPath) // platform-specific, defined in daemon_windows.go / daemon_unix.go
		return
	}

	if *stop {
		stopDaemon()
		return
	}
	// ==============================================

	// Load config
	cfg, err := loadConfig(*envPath)
	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}

	// Setup logger
	logger := logger.NewLogger(cfg.LogLevel)

	// Setup storage
	store, err := storage.NewStorage("monitor.db")
	if err != nil {
		logger.Fatal("Failed to init storage", "error", err)
	}
	defer store.Close()

	// Setup network client
	client := network.NewClient(cfg.ServerURL, cfg.APIKey, cfg.BatchSize)

	handler := newSearchHandler(cfg, store, client, logger)
	searchServer := &http.Server{
		Addr:              searchReceiverAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", searchReceiverAddress)
	if err != nil {
		logger.Fatal("Failed to start browser search receiver", "error", err)
	}
	go func() {
		if err := searchServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Error("Browser search receiver stopped", "error", err)
		}
	}()
	logger.Info("Parental Monitor CLI started", "device", cfg.DeviceName)
	logger.Info("Browser search receiver listening", "address", searchReceiverAddress)

	// Start background flush of queued data
	client.StartFlusher(cfg.FlushIntervalSec)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := searchServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to stop browser search receiver", "error", err)
	}
	client.Flush()
	store.Close()
}

func newSearchHandler(cfg Config, store *storage.Storage, client *network.Client, logger *logger.Logger) http.Handler {

	keywords := append(append([]string(nil), cfg.ScanKeywords...), cfg.CustomKeywords...)
	keywordGroups := monitor.MergeKeywordGroups(cfg.ScanKeywordGroups, cfg.CustomKeywordGroups)
	analyzer := monitor.NewGroupedAnalyzer(keywords, keywordGroups)

	return searchreceiver.NewHandler(
		cfg.DeviceName,
		analyzer,
		func(query monitor.SearchEvent) {
			query.ChildID = cfg.ChildID
			query.DeviceID = cfg.DeviceID
			logger.Info("Browser search received", "engine", query.Engine, "incognito", query.Incognito, "query_length", len(query.Query))

			if store != nil {
				if err := store.SaveSearch(query); err != nil {
					logger.Error("Failed to save search", "error", err, "query", query.Query)
				}
			}
		},
		func(alert monitor.AlertEvent) {
			alert.ChildID = cfg.ChildID
			alert.DeviceID = cfg.DeviceID
			logger.Warn("Keyword alert detected", "category", alert.Category, "keyword", alert.Keyword)
			if store != nil {
				if err := store.SaveAlert(alert); err != nil {
					logger.Error("Failed to save alert", "error", err, "query", alert.Query)
				}
			}
			if client != nil {
				client.QueueAlert(alert)
			}
		},
	)
}
