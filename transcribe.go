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

// errHTTP marks a response from the server, which retrying will not fix.
type errHTTP struct{ msg string }

func (e errHTTP) Error() string { return e.msg }

// transcribe retries network failures (resets, timeouts) up to three times.
func transcribe(cfg Config, path string) (string, error) {
	if cfg.APIKey == "" {
		return "", errors.New("no API key: set OPENAI_API_KEY or openai_api_key in " + configPath())
	}
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		var text string
		text, err = transcribeOnce(cfg, path)
		var httpErr errHTTP
		if err == nil || errors.As(err, &httpErr) {
			return text, err
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return "", err
}

// transcribeOnce sends the audio to an OpenAI-compatible /audio/transcriptions endpoint.
func transcribeOnce(cfg Config, path string) (string, error) {
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
		// gpt-transcribe takes a languages list and rejects the singular field.
		if cfg.Model == "gpt-transcribe" {
			w.WriteField("languages[]", cfg.Language)
		} else {
			w.WriteField("language", cfg.Language)
		}
	}
	if cfg.Prompt != "" {
		w.WriteField("prompt", cfg.Prompt)
	}
	w.Close()

	// Allow 60 s for processing plus 1 s per 2 KB uploaded, for slow links.
	timeout := 60*time.Second + time.Duration(body.Len()/2000)*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errHTTP{fmt.Sprintf("transcription failed (%s): %s", resp.Status, strings.TrimSpace(string(data)))}
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("bad transcription response: %w", err)
	}
	// A typed newline is Enter, which would submit chat boxes mid-dictation.
	return strings.Join(strings.Fields(out.Text), " "), nil
}
