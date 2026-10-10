package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// minSeconds skips accidental taps that captured no speech.
const minSeconds = 0.3

func toggle() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	var wav string
	err = withLock(func() error {
		if pid := runningRecorder(); pid != 0 {
			if err := stopRecording(pid); err != nil {
				return err
			}
			// Move the file aside so a new recording can start while we transcribe.
			wav = filepath.Join(stateDir(), fmt.Sprintf("recording-%d.wav", os.Getpid()))
			return os.Rename(wavFile(), wav)
		}
		if err := startRecording(); err != nil {
			return err
		}
		if cfg.notifyEnabled() {
			notify("Recording…", "Press the shortcut again to stop")
		}
		return nil
	})
	if err != nil || wav == "" {
		return err
	}
	defer os.Remove(wav)

	entry := historyEntry{Time: time.Now(), Seconds: wavDuration(wav)}
	err = transcribeAndEmit(cfg, wav, &entry)
	if err != nil {
		entry.Error = err.Error()
	}
	appendHistory(entry)
	return err
}

func transcribeAndEmit(cfg Config, wav string, entry *historyEntry) error {
	if entry.Seconds < minSeconds {
		if cfg.notifyEnabled() {
			notify("Too short", "Nothing recorded")
		}
		return nil
	}
	if cfg.notifyEnabled() {
		notify("Transcribing…", "")
	}
	upload := compress(wav)
	if upload != wav {
		defer os.Remove(upload)
	}
	text, err := transcribe(cfg, upload)
	if err != nil {
		return err
	}
	entry.Raw = text
	if text == "" {
		if cfg.notifyEnabled() {
			notify("No speech detected", "")
		}
		return nil
	}
	text = runHook(cfg, text)
	entry.Typed = text
	if cfg.notifyEnabled() {
		notify("Done", fmt.Sprintf("%d chars", len(text)))
	}
	return emit(cfg, text)
}

func cancel() error {
	return withLock(func() error {
		if pid := runningRecorder(); pid != 0 {
			stopRecording(pid)
			os.Remove(wavFile())
			notify("Cancelled", "")
		}
		return nil
	})
}

func transcribeFile(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: gowhspr transcribe <file>")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	text, err := transcribe(cfg, args[0])
	if err != nil {
		return err
	}
	fmt.Println(text)
	return nil
}

func status() error {
	if runningRecorder() != 0 {
		fmt.Println("recording")
	} else {
		fmt.Println("idle")
	}
	return nil
}
