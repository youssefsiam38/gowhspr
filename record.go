package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Recording state lives in $XDG_RUNTIME_DIR/gowhspr: the recorder's PID and
// the WAV it writes. Each shortcut press is a separate short-lived process.

func stateDir() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "gowhspr")
}

func pidFile() string { return filepath.Join(stateDir(), "recorder.pid") }
func wavFile() string { return filepath.Join(stateDir(), "recording.wav") }

// withLock serialises toggle/cancel so a double press can't start two recorders.
func withLock(fn func() error) error {
	if err := os.MkdirAll(stateDir(), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(stateDir(), "lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

// runningRecorder returns the PID of a live recorder, or 0.
func runningRecorder() int {
	data, err := os.ReadFile(pidFile())
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0
	}
	if syscall.Kill(pid, 0) != nil {
		os.Remove(pidFile())
		return 0
	}
	return pid
}

// startRecording launches a detached recorder on the system default source.
// No target is passed, so PipeWire/PulseAudio routes whatever mic is
// currently selected in system settings.
func startRecording() error {
	os.Remove(wavFile())
	name, args := recorderCommand(wavFile())
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}
	pid := cmd.Process.Pid
	cmd.Process.Release()
	return os.WriteFile(pidFile(), []byte(strconv.Itoa(pid)), 0o600)
}

func recorderCommand(out string) (string, []string) {
	if _, err := exec.LookPath("pw-record"); err == nil {
		return "pw-record", []string{"--rate", "16000", "--channels", "1", "--format", "s16", out}
	}
	return "parecord", []string{"--rate=16000", "--channels=1", "--format=s16le", "--file-format=wav", out}
}

// stopRecording signals the recorder and waits for it to finalise the WAV.
func stopRecording(pid int) error {
	defer os.Remove(pidFile())
	if err := syscall.Kill(pid, syscall.SIGINT); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	syscall.Kill(pid, syscall.SIGKILL)
	return nil
}

// compress re-encodes the WAV as 24 kbps Opus (~11x smaller) so uploads stay
// fast on slow links. Without ffmpeg, or if encoding fails, the WAV is used.
func compress(wav string) string {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return wav
	}
	ogg := strings.TrimSuffix(wav, ".wav") + ".ogg"
	cmd := exec.Command("ffmpeg", "-loglevel", "error", "-y", "-i", wav,
		"-c:a", "libopus", "-b:a", "24k", "-application", "voip", ogg)
	if err := cmd.Run(); err != nil {
		os.Remove(ogg)
		return wav
	}
	return ogg
}

// wavDuration estimates seconds of 16 kHz mono s16 audio from file size.
func wavDuration(path string) float64 {
	st, err := os.Stat(path)
	if err != nil || st.Size() <= 44 {
		return 0
	}
	return float64(st.Size()-44) / (16000 * 2)
}
