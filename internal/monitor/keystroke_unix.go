//go:build !windows

package monitor

import (
	"bufio"
	"os"
	"sync/atomic"
	"time"
)

type KeystrokeMonitor struct {
	keys     chan rune
	active   atomic.Bool
	stopChan chan struct{}
}

func NewKeystrokeMonitor() *KeystrokeMonitor {
	return &KeystrokeMonitor{
		keys:     make(chan rune, 1024),
		stopChan: make(chan struct{}),
	}
}

func (k *KeystrokeMonitor) Start() error {
	go k.readFromDevice()
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

// Note: On Linux, you'd read from /dev/input/eventX
// This is a simplified version that reads from stdin as fallback
func (k *KeystrokeMonitor) readFromDevice() {
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-k.stopChan:
			return
		default:
			if !k.active.Load() {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			r, _, err := reader.ReadRune()
			if err == nil {
				select {
				case k.keys <- r:
				default:
				}
			}
		}
	}
}
