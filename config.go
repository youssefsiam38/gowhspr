package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey   string `json:"openai_api_key,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
	Model    string `json:"model,omitempty"`
	Language string `json:"language,omitempty"`
	Prompt   string `json:"prompt,omitempty"`
	// Output is "type" (ydotool keystrokes) or "paste" (clipboard + Ctrl+V).
	Output string `json:"output,omitempty"`
	// PostHook receives the transcript on stdin; non-empty stdout replaces it.
	PostHook string `json:"post_transcription_hook,omitempty"`
	// Notify toggles desktop notifications for recording state.
	Notify *bool `json:"notify,omitempty"`
}

func configPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "gowhspr", "config.json")
}

func loadConfig() (Config, error) {
	cfg := Config{}
	data, err := os.ReadFile(configPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return cfg, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse %s: %w", configPath(), err)
		}
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		cfg.APIKey = k
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-transcribe"
	}
	if cfg.Output == "" {
		cfg.Output = "type"
	}
	return cfg, nil
}

func (c Config) notifyEnabled() bool { return c.Notify == nil || *c.Notify }

func expandHome(p string) string {
	if len(p) > 1 && p[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func showConfig() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	key := "(missing)"
	if cfg.APIKey != "" {
		key = "(set)"
	}
	fmt.Printf("config:   %s\napi key:  %s\nbase url: %s\nmodel:    %s\nlanguage: %s\noutput:   %s\nhook:     %s\n",
		configPath(), key, cfg.BaseURL, cfg.Model, cfg.Language, cfg.Output, cfg.PostHook)
	return nil
}
