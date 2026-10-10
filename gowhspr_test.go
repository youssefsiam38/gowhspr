package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestASCIINormalisation(t *testing.T) {
	got := asciiReplacer.Replace("I’m “done” — really…")
	if want := `I'm "done" - really...`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if !isASCII(got) {
		t.Fatal("normalised text should be ASCII")
	}
	if isASCII("café") {
		t.Fatal("accented text is not ASCII")
	}
}

func TestTranscribeRetriesNetworkErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			// Drop the connection, like the reset seen in production.
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
			return
		}
		w.Write([]byte(`{"text":" hello world "}`))
	}))
	defer srv.Close()

	audio := filepath.Join(t.TempDir(), "a.ogg")
	os.WriteFile(audio, []byte("audio"), 0o600)
	text, err := transcribe(Config{APIKey: "k", BaseURL: srv.URL, Model: "m"}, audio)
	if err != nil || text != "hello world" || calls.Load() != 2 {
		t.Fatalf("text=%q err=%v calls=%d", text, err, calls.Load())
	}
}

func TestTranscribeDoesNotRetryHTTPErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, `{"error":"bad key"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	audio := filepath.Join(t.TempDir(), "a.ogg")
	os.WriteFile(audio, []byte("audio"), 0o600)
	if _, err := transcribe(Config{APIKey: "k", BaseURL: srv.URL, Model: "m"}, audio); err == nil || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestHistoryIsBounded(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for i := 0; i < maxHistory+5; i++ {
		appendHistory(historyEntry{Raw: "x"})
	}
	data, _ := os.ReadFile(historyPath())
	if n := len(splitLines(data)); n != maxHistory {
		t.Fatalf("got %d entries, want %d", n, maxHistory)
	}
}
