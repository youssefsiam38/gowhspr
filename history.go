package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// historyEntry is one dictation, appended to history.jsonl for troubleshooting.
type historyEntry struct {
	Time    time.Time `json:"time"`
	Seconds float64   `json:"seconds"`
	Raw     string    `json:"transcript,omitempty"`
	Typed   string    `json:"typed,omitempty"`
	Error   string    `json:"error,omitempty"`
}

func historyPath() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "gowhspr", "history.jsonl")
}

// maxHistory bounds the log; the oldest entries are dropped past it.
const maxHistory = 200

func appendHistory(e historyEntry) {
	path := historyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	line, err := json.Marshal(e)
	if err != nil {
		return
	}
	data, _ := os.ReadFile(path)
	lines := splitLines(data)
	lines = append(lines, line)
	if len(lines) > maxHistory {
		lines = lines[len(lines)-maxHistory:]
	}
	var out []byte
	for _, l := range lines {
		out = append(append(out, l...), '\n')
	}
	os.WriteFile(path, out, 0o600)
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// keepLastRecording copies the uploaded audio next to the history log,
// replacing the previous one, so a bad transcript can be checked against it.
func keepLastRecording(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	dir := filepath.Dir(historyPath())
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	for _, old := range []string{"last.ogg", "last.webm", "last.wav"} {
		os.Remove(filepath.Join(dir, old))
	}
	os.WriteFile(filepath.Join(dir, "last"+filepath.Ext(path)), data, 0o600)
}
