package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// transcribe sends the WAV to an OpenAI-compatible /audio/transcriptions endpoint.
func transcribe(cfg Config, path string) (string, error) {
	if cfg.APIKey == "" {
		return "", errors.New("no API key: set OPENAI_API_KEY or openai_api_key in " + configPath())
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	w.WriteField("model", cfg.Model)
	w.WriteField("response_format", "json")
	if cfg.Language != "" {
		w.WriteField("language", cfg.Language)
	}
	if cfg.Prompt != "" {
		w.WriteField("prompt", cfg.Prompt)
	}
	w.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	url := strings.TrimRight(cfg.BaseURL, "/") + "/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("transcription failed (%s): %s", resp.Status, strings.TrimSpace(string(data)))
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("bad transcription response: %w", err)
	}
	return strings.TrimSpace(out.Text), nil
}
