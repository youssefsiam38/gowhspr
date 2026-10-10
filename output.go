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

// asciiReplacer maps typographic characters models emit to keys ydotool can type.
var asciiReplacer = strings.NewReplacer(
	"\u2018", "'", "\u2019", "'", "\u201c", `"`, "\u201d", `"`,
	"\u2013", "-", "\u2014", "-", "\u2026", "...", "\u00a0", " ",
)

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// typeText types ASCII with ydotool, which silently drops characters it has
// no key for; anything still non-ASCII after normalising is pasted instead.
func typeText(text string) error {
	text = asciiReplacer.Replace(text)
	if !isASCII(text) {
		return paste(text)
	}
	cmd := exec.Command("ydotool", "type", "--key-delay=12", "--key-hold=8", "--file=-")
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
