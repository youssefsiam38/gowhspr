// gowhspr: push-to-toggle dictation for Linux desktops.
//
// Press the shortcut once to start recording from the system default
// microphone, press it again to stop; the audio is transcribed and typed
// into the focused window.
package main

import (
	"fmt"
	"os"
)

const usage = `gowhspr - dictation for Linux (records the system default mic)

Usage:
  gowhspr toggle              start recording, or stop and type the transcript
  gowhspr cancel              stop recording and discard it
  gowhspr status              print "recording" or "idle"
  gowhspr transcribe <file>   transcribe an audio file and print the text
  gowhspr shortcut [flags]    register GNOME keyboard shortcuts
      --key <accel>           toggle shortcut (default "<Super><Alt>d")
      --cancel-key <accel>    cancel shortcut (default "<Super><Alt>x")
      --remove                remove the shortcuts instead
  gowhspr config              print the config file path and current settings

Config: ~/.config/gowhspr/config.json (API key may also come from OPENAI_API_KEY)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "toggle":
		err = toggle()
	case "cancel":
		err = cancel()
	case "status":
		err = status()
	case "transcribe":
		err = transcribeFile(os.Args[2:])
	case "shortcut":
		err = shortcut(os.Args[2:])
	case "config":
		err = showConfig()
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gowhspr:", err)
		notify("gowhspr error", err.Error())
		os.Exit(1)
	}
}
