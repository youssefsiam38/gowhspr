# gowhspr

Dictation for Linux as one small Go binary. Press a shortcut to start recording, press it again to stop: the audio is transcribed and typed into the focused window.

- **Always uses the system default microphone.** Recording goes through `pw-record` (or `parecord`) with no device pinned, so whatever mic is selected in system settings is what gets recorded, including mics plugged in after login.
- **No daemon.** Each shortcut press runs a short-lived process; state is a PID file in `$XDG_RUNTIME_DIR`.
- **Whole-clip transcription.** The finished recording is sent to an OpenAI-compatible `/audio/transcriptions` endpoint, so no audio is dropped by a streaming connection.
- **Built-in shortcut setup** for GNOME (Wayland or X11).

## Requirements

- PipeWire (`pw-record`) or PulseAudio (`parecord`)
- [`ydotool`](https://github.com/ReimuNotMoe/ydotool) with `ydotoold` running (types the text)
- `wl-copy` (only for `"output": "paste"`)
- `ffmpeg` with libopus (optional, recommended: compresses audio ~11x before upload; falls back to WAV)
- `notify-send` (optional, for status notifications)
- An OpenAI API key, or any OpenAI-compatible transcription server

## Install

```bash
go install github.com/youssefsiam38/gowhspr@latest
```

or from a clone:

```bash
make install   # builds and copies to ~/.local/bin/gowhspr
```

## Setup

```bash
mkdir -p ~/.config/gowhspr
printf '{\n  "openai_api_key": "sk-..."\n}\n' > ~/.config/gowhspr/config.json
chmod 600 ~/.config/gowhspr/config.json

gowhspr shortcut          # Super+Alt+D toggles, Super+Alt+X cancels
```

Custom keys: `gowhspr shortcut --key '<Control><Alt>space' --cancel-key '<Control><Alt>c'`. Remove with `gowhspr shortcut --remove`.

On other desktops (KDE, Hyprland, Sway, ...) bind `gowhspr toggle` and `gowhspr cancel` in your compositor's shortcut settings.

## Commands

| Command | Action |
|---|---|
| `gowhspr toggle` | Start recording, or stop and type the transcript |
| `gowhspr cancel` | Stop recording and discard it |
| `gowhspr status` | Print `recording` or `idle` |
| `gowhspr transcribe <file>` | Transcribe an audio file and print the text |
| `gowhspr shortcut` | Register GNOME shortcuts |
| `gowhspr config` | Show the config in use |

## Config

`~/.config/gowhspr/config.json`. All fields are optional except the API key (which can also come from `OPENAI_API_KEY`).

| Field | Default | Meaning |
|---|---|---|
| `openai_api_key` | | API key |
| `base_url` | `https://api.openai.com/v1` | Any OpenAI-compatible server (Groq, local whisper servers, ...) |
| `model` | `gpt-4o-transcribe` | Transcription model |
| `language` | auto | ISO-639-1 code, e.g. `en` |
| `prompt` | | Vocabulary or style hint for the model |
| `output` | `type` | `type` (ydotool keystrokes) or `paste` (clipboard + Ctrl+V) |
| `post_transcription_hook` | | Program that gets the transcript on stdin; non-empty stdout replaces it (5 s timeout) |
| `notify` | `true` | Desktop notifications |

## Troubleshooting

- **Nothing is typed:** check `ydotoold` is running and your user can reach its socket (`ydotool type hello`).
- **Wrong mic:** change the input device in system sound settings; the next recording uses it.
- **Test the API without the mic:** `gowhspr transcribe some.wav`.

## License

MIT
