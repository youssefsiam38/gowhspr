package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// runHook pipes text through the user's post-transcription hook. Empty
// output, errors, or a timeout leave the transcript unchanged.
func runHook(cfg Config, text string) string {
	if cfg.PostHook == "" {
		return text
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, expandHome(cfg.PostHook))
	cmd.Stdin = strings.NewReader(text)
	out, err := cmd.Output()
	if err != nil {
		return text
	}
	if s := strings.TrimSpace(string(out)); s != "" {
		return s
	}
	return text
}

// emit delivers text to the focused window.
func emit(cfg Config, text string) error {
	if cfg.Output == "paste" {
		return paste(text)
	}
	return typeText(text)
}

func typeText(text string) error {
	cmd := exec.Command("ydotool", "type", "--key-delay=4", "--key-hold=4", "--file=-")
	cmd.Stdin = strings.NewReader(text)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ydotool type: %v %s (is ydotoold running?)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func paste(text string) error {
	cp := exec.Command("wl-copy")
	cp.Stdin = strings.NewReader(text)
	if err := cp.Run(); err != nil {
		return fmt.Errorf("wl-copy: %w", err)
	}
	time.Sleep(80 * time.Millisecond)
	// Ctrl down, V down, V up, Ctrl up (Linux input keycodes 29 and 47).
	if out, err := exec.Command("ydotool", "key", "29:1", "47:1", "47:0", "29:0").CombinedOutput(); err != nil {
		return fmt.Errorf("ydotool key: %v %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func notify(title, body string) {
	exec.Command("notify-send", "-a", "gowhspr", "-t", "2000",
		"-h", "string:x-canonical-private-synchronous:gowhspr", title, body).Run()
}
