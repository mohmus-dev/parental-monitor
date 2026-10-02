//go:build windows

package monitor

import (
	"sync/atomic"
	"syscall"
	"time"
)

// Windows API
var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
)

type KeystrokeMonitor struct {
	keys     chan rune
	active   atomic.Bool
	stopChan chan struct{}
	keyMap   map[int]rune
}

func NewKeystrokeMonitor() *KeystrokeMonitor {
	return &KeystrokeMonitor{
		keys:     make(chan rune, 1024),
		stopChan: make(chan struct{}),
		keyMap:   buildKeyMap(),
	}
}

func (k *KeystrokeMonitor) Start() error {
	go k.poll()
	return nil
}

func (k *KeystrokeMonitor) Stop() {
	close(k.stopChan)
}

func (k *KeystrokeMonitor) SetActive(active bool) {
	k.active.Store(active)
}

func (k *KeystrokeMonitor) Keys() <-chan rune {
	return k.keys
}

func (k *KeystrokeMonitor) poll() {
	// Track previous state to detect key presses (not just state)
	prevState := make(map[int]bool)

	for {
		select {
		case <-k.stopChan:
			return
		default:
			if !k.active.Load() {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Poll all virtual key codes
			for vk := 0x08; vk <= 0xFE; vk++ {
				state, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
				pressed := state&0x8000 != 0

				if pressed && !prevState[vk] {
					// Key was just pressed
					if r, ok := k.keyMap[vk]; ok {
						select {
						case k.keys <- r:
						default:
							// Channel full, skip
						}
					}
				}
				prevState[vk] = pressed
			}

			time.Sleep(2 * time.Millisecond)
		}
	}
}

func buildKeyMap() map[int]rune {
	m := make(map[int]rune)

	// Letters
	for i := 0x41; i <= 0x5A; i++ {
		m[i] = rune('a' + i - 0x41)
	}

	// Numbers
	for i := 0x30; i <= 0x39; i++ {
		m[i] = rune('0' + i - 0x30)
	}

	// Special keys
	m[0x08] = 8    // Backspace
	m[0x0D] = '\r' // Enter
	m[0x1B] = 27   // Escape
	m[0x20] = ' '  // Space

	// Punctuation
	m[0xBA] = ';'
	m[0xBB] = '='
	m[0xBC] = ','
	m[0xBD] = '-'
	m[0xBE] = '.'
	m[0xBF] = '/'
	m[0xC0] = '`'
	m[0xDB] = '['
	m[0xDC] = '\\'
	m[0xDD] = ']'
	m[0xDE] = '\''

	return m
}
