package monitor

import (
	"fmt"
	"sync"
	"time"
)

type SearchEvent struct {
	Query       string    `json:"query"`
	Engine      string    `json:"engine"`
	Timestamp   time.Time `json:"timestamp"`
	DeviceName  string    `json:"device_name"`
	Incognito   bool      `json:"incognito"`
	WindowTitle string    `json:"window_title"`
	URL         string    `json:"url"`
}

type AlertEvent struct {
	Category  string    `json:"category,omitempty"`
	Keyword   string    `json:"keyword"`
	Query     string    `json:"query"`
	Timestamp time.Time `json:"timestamp"`
}

type Config struct {
	IntervalMs    int
	Keywords      []string
	KeywordGroups map[string][]string
	DeviceName    string
}

type Monitor struct {
	cfg         Config
	keystroke   *KeystrokeMonitor
	window      *WindowMonitor
	analyzer    *Analyzer
	onSearch    func(SearchEvent)
	onAlert     func(AlertEvent)
	stopChan    chan struct{}
	mu          sync.Mutex
	buffer      []rune
	lastKeyTime time.Time
	running     bool
}

func NewMonitor(cfg Config) *Monitor {
	return &Monitor{
		cfg:       cfg,
		keystroke: NewKeystrokeMonitor(),
		window:    NewWindowMonitor(),
		analyzer:  NewGroupedAnalyzer(cfg.Keywords, cfg.KeywordGroups),
		stopChan:  make(chan struct{}),
		buffer:    make([]rune, 0, 256),
	}
}

func (m *Monitor) OnSearchQuery(fn func(SearchEvent)) {
	m.onSearch = fn
}

func (m *Monitor) OnKeywordAlert(fn func(AlertEvent)) {
	m.onAlert = fn
}

func (m *Monitor) Start() error {
	if m.running {
		return nil
	}
	m.running = true

	// Start keystroke listener
	if err := m.keystroke.Start(); err != nil {
		return err
	}

	// Start window polling goroutine
	go m.windowLoop()

	// Start keystroke processing goroutine
	go m.processKeys()

	return nil
}

func (m *Monitor) Stop() {
	if !m.running {
		return
	}
	m.running = false
	close(m.stopChan)
	m.keystroke.Stop()
}

func (m *Monitor) windowLoop() {
	ticker := time.NewTicker(time.Duration(m.cfg.IntervalMs) * time.Millisecond)
	defer ticker.Stop()
	wasActive := false

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			title := m.window.GetActiveWindowTitle()
			active := isChromeWindow(title)
			m.keystroke.SetActive(active)
			if active != wasActive {
				fmt.Printf("[TRACE] Chrome key capture active=%t\n", active)
				wasActive = active
			}
		}
	}
}

func (m *Monitor) processKeys() {
	for {
		select {
		case <-m.stopChan:
			return
		case key := <-m.keystroke.Keys():
			fmt.Printf("[TRACE] Key event received: %s\n", keyEventType(key))
			m.handleKey(key)
		}
	}
}

func (m *Monitor) handleKey(key rune) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Reset buffer if too much time passed (user stopped typing)
	if !m.lastKeyTime.IsZero() && now.Sub(m.lastKeyTime) > 3*time.Second {
		m.buffer = m.buffer[:0]
	}

	m.lastKeyTime = now

	// Handle special keys
	switch key {
	case '\r', '\n':
		fmt.Printf("[TRACE] Enter received; buffered characters=%d\n", len(m.buffer))
		m.processBuffer()
		m.buffer = m.buffer[:0]
		return
	case 8: // Backspace
		if len(m.buffer) > 0 {
			m.buffer = m.buffer[:len(m.buffer)-1]
		}
		return
	case 27: // Escape
		m.buffer = m.buffer[:0]
		return
	}

	// Only capture printable characters
	if key >= 32 && key <= 126 {
		if len(m.buffer) < 256 {
			m.buffer = append(m.buffer, key)
		}
	}
}

func (m *Monitor) processBuffer() {
	if len(m.buffer) < 2 {
		fmt.Printf("[TRACE] Submission ignored; buffered characters=%d\n", len(m.buffer))
		return
	}

	query := string(m.buffer)
	windowTitle := m.window.GetActiveWindowTitle()
	engine := detectEngine(windowTitle)

	// Create search event
	event := SearchEvent{
		Query:       query,
		Engine:      engine,
		Timestamp:   time.Now(),
		DeviceName:  m.cfg.DeviceName,
		Incognito:   isIncognito(windowTitle),
		WindowTitle: windowTitle,
	}

	// Fire callback
	if m.onSearch != nil {
		m.onSearch(event)
	}

	matches := m.analyzer.FindMatches(query)
	fmt.Printf("[TRACE] Keyword categories matched=%d\n", len(matches))
	for _, match := range matches {
		fmt.Printf("[TRACE] Keyword match category=%s keyword=%s\n", match.Category, match.Keyword)
	}
	if m.onAlert != nil {
		for _, match := range matches {
			m.onAlert(AlertEvent{
				Category:  match.Category,
				Keyword:   match.Keyword,
				Query:     query,
				Timestamp: time.Now(),
			})
		}
	}
}

func keyEventType(key rune) string {
	switch key {
	case '\r', '\n':
		return "Enter"
	case 8:
		return "Backspace"
	case 27:
		return "Escape"
	default:
		if key >= 32 && key <= 126 {
			return "printable"
		}
		return "unsupported"
	}
}

func detectEngine(windowTitle string) string {
	lower := toLower(windowTitle)
	switch {
	case contains(lower, "google"):
		return "Google"
	case contains(lower, "youtube"):
		return "YouTube"
	case contains(lower, "bing"):
		return "Bing"
	case contains(lower, "duckduckgo"):
		return "DuckDuckGo"
	default:
		return "Unknown"
	}
}

func isIncognito(windowTitle string) bool {
	return contains(toLower(windowTitle), "incognito")
}

func isChromeWindow(title string) bool {
	lower := toLower(title)
	return contains(lower, "chrome") || contains(lower, "google chrome")
}

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && indexOf(s, substr) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}
