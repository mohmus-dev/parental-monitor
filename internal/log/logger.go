package log

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

type Logger struct {
	mu    sync.Mutex
	level Level
	file  *os.File
}

func NewLogger(levelStr string) *Logger {
	level := INFO
	switch strings.ToLower(levelStr) {
	case "debug":
		level = DEBUG
	case "warn":
		level = WARN
	case "error":
		level = ERROR
	}

	// Optional file logging
	file, _ := os.OpenFile("monitor.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	return &Logger{
		level: level,
		file:  file,
	}
}

func (l *Logger) log(level Level, prefix string, msg string, kv ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	ts := time.Now().Format("2006-01-02 15:04:05")
	parts := make([]string, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			parts = append(parts, fmt.Sprintf("%s=%v", kv[i], kv[i+1]))
		}
	}

	line := fmt.Sprintf("[%s] %s %s %s\n", ts, prefix, msg, strings.Join(parts, " "))
	fmt.Print(line)
	if l.file != nil {
		l.file.WriteString(line)
	}
}

func (l *Logger) Debug(msg string, kv ...interface{}) { l.log(DEBUG, "DEBUG", msg, kv...) }
func (l *Logger) Info(msg string, kv ...interface{})  { l.log(INFO, "INFO", msg, kv...) }
func (l *Logger) Warn(msg string, kv ...interface{})  { l.log(WARN, "WARN", msg, kv...) }
func (l *Logger) Error(msg string, kv ...interface{}) { l.log(ERROR, "ERROR", msg, kv...) }
func (l *Logger) Fatal(msg string, kv ...interface{}) {
	l.log(FATAL, "FATAL", msg, kv...)
	os.Exit(1)
}
