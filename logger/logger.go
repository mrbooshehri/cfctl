package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Level string

const (
	LevelInfo    Level = "INFO"
	LevelSuccess Level = "SUCCESS"
	LevelWarn    Level = "WARN"
	LevelError   Level = "ERROR"
)

type Entry struct {
	Time    time.Time `json:"time"`
	Level   Level     `json:"level"`
	Section string    `json:"section"`
	Zone    string    `json:"zone"`
	Message string    `json:"message"`
}

type Logger struct {
	path string
}

func New() (*Logger, error) {
	base, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(base, ".local", "share", "cfctl")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Logger{path: filepath.Join(dir, "activity.log")}, nil
}

func (l *Logger) Write(level Level, section, zone, message string) {
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	e := Entry{Time: time.Now(), Level: level, Section: section, Zone: zone, Message: message}
	_ = json.NewEncoder(f).Encode(e)
}

func (l *Logger) Read() ([]Entry, error) {
	data, err := os.ReadFile(l.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var e Entry
		if json.Unmarshal(line, &e) == nil {
			entries = append(entries, e)
		}
	}

	// Reverse so newest entries are first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

func (l *Logger) Clear() error {
	return os.WriteFile(l.path, nil, 0600)
}

func (l *Logger) Path() string { return l.path }
