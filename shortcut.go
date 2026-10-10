package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	mediaKeys  = "org.gnome.settings-daemon.plugins.media-keys"
	customKey  = mediaKeys + ".custom-keybinding"
	bindingDir = "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/"
)

// shortcut registers (or removes) GNOME custom keybindings that run
// `gowhspr toggle` and `gowhspr cancel`.
func shortcut(args []string) error {
	fs := flag.NewFlagSet("shortcut", flag.ContinueOnError)
	key := fs.String("key", "<Super><Alt>d", "toggle shortcut")
	cancelKey := fs.String("cancel-key", "<Super><Alt>x", "cancel shortcut")
	remove := fs.Bool("remove", false, "remove the shortcuts")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if _, err := exec.LookPath("gsettings"); err != nil {
		return errors.New("gsettings not found: only GNOME is supported; bind `gowhspr toggle` in your desktop's shortcut settings")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.EvalSymlinks(exe)

	bindings := []struct{ id, name, key, cmd string }{
		{"gowhspr-toggle", "gowhspr dictation", *key, exe + " toggle"},
		{"gowhspr-cancel", "gowhspr cancel", *cancelKey, exe + " cancel"},
	}

	list, err := customBindings()
	if err != nil {
		return err
	}
	for _, b := range bindings {
		path := bindingDir + b.id + "/"
		list = without(list, path)
		if *remove {
			exec.Command("gsettings", "reset-recursively", customKey+":"+path).Run()
			continue
		}
		list = append(list, path)
		for k, v := range map[string]string{"name": b.name, "binding": b.key, "command": b.cmd} {
			if out, err := exec.Command("gsettings", "set", customKey+":"+path, k, v).CombinedOutput(); err != nil {
				return fmt.Errorf("gsettings set %s: %v %s", k, err, out)
			}
		}
	}
	if out, err := exec.Command("gsettings", "set", mediaKeys, "custom-keybindings", formatList(list)).CombinedOutput(); err != nil {
		return fmt.Errorf("gsettings: %v %s", err, out)
	}
	if *remove {
		fmt.Println("Removed gowhspr shortcuts.")
	} else {
		fmt.Printf("Registered %s → toggle, %s → cancel\n", *key, *cancelKey)
	}
	return nil
}

func customBindings() ([]string, error) {
	out, err := exec.Command("gsettings", "get", mediaKeys, "custom-keybindings").Output()
	if err != nil {
		return nil, fmt.Errorf("gsettings get: %w", err)
	}
	s := strings.TrimSpace(string(out))
	s = strings.TrimPrefix(s, "@as ")
	s = strings.Trim(s, "[]")
	var list []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.Trim(strings.TrimSpace(p), "'"); p != "" {
			list = append(list, p)
		}
	}
	return list, nil
}

func without(list []string, item string) []string {
	out := list[:0]
	for _, v := range list {
		if v != item {
			out = append(out, v)
		}
	}
	return out
}

func formatList(list []string) string {
	quoted := make([]string, len(list))
	for i, v := range list {
		quoted[i] = "'" + v + "'"
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
