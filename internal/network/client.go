package network

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"parental-monitor-cli/internal/monitor"
)

type Client struct {
	serverURL string
	apiKey    string
	batchSize int

	mu       sync.Mutex
	searches []monitor.SearchEvent
	alerts   []monitor.AlertEvent

	httpClient *http.Client
}

func NewClient(serverURL, apiKey string, batchSize int) *Client {
	return &Client{
		serverURL: serverURL,
		apiKey:    apiKey,
		batchSize: batchSize,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) QueueSearch(event monitor.SearchEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.searches = append(c.searches, event)

	if len(c.searches) >= c.batchSize {
		go c.sendBatch()
	}
}

func (c *Client) QueueAlert(event monitor.AlertEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.alerts = append(c.alerts, event)
	go c.sendAlerts()
}

func (c *Client) StartFlusher(intervalSec int) {
	go func() {
		ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			c.Flush()
		}
	}()
}

func (c *Client) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.searches) > 0 {
		go c.sendBatch()
	}
	if len(c.alerts) > 0 {
		go c.sendAlerts()
	}
}

func (c *Client) sendBatch() {
	c.mu.Lock()
	searches := c.searches
	c.searches = nil
	c.mu.Unlock()

	if len(searches) == 0 {
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"type":     "searches",
		"searches": searches,
	})

	req, _ := http.NewRequest("POST", c.serverURL+"/api/ingest", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Queue back for retry
		c.mu.Lock()
		c.searches = append(searches, c.searches...)
		c.mu.Unlock()
		return
	}
	defer resp.Body.Close()
}

func (c *Client) sendAlerts() {
	c.mu.Lock()
	alerts := c.alerts
	c.alerts = nil
	c.mu.Unlock()

	if len(alerts) == 0 {
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"type":   "alerts",
		"alerts": alerts,
	})
	if err != nil {
		log.Printf("keyword alert delivery failed: could not encode payload: %v", err)
		c.requeueAlerts(alerts)
		return
	}

	req, err := http.NewRequest("POST", c.serverURL+"/api/alerts", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("keyword alert delivery failed: could not create request: %v", err)
		c.requeueAlerts(alerts)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("keyword alert delivery failed: %v", err)
		c.requeueAlerts(alerts)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("keyword alert delivery rejected: status=%s count=%d", resp.Status, len(alerts))
		c.requeueAlerts(alerts)
		return
	}

	log.Printf("keyword alert batch delivered: status=%s count=%d", resp.Status, len(alerts))
}

func (c *Client) requeueAlerts(alerts []monitor.AlertEvent) {
	c.mu.Lock()
	c.alerts = append(alerts, c.alerts...)
	c.mu.Unlock()
}
