//go:build windows

package monitor

import (
	"testing"
	"time"
)

func TestSetActiveWhilePolling(t *testing.T) {
	keystroke := NewKeystrokeMonitor()
	if err := keystroke.Start(); err != nil {
		t.Fatal(err)
	}
	defer keystroke.Stop()

	for i := 0; i < 100; i++ {
		keystroke.SetActive(i%2 == 0)
		time.Sleep(time.Millisecond)
	}
}
