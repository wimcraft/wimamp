package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaultKeys maps TUI actions to space-separated keys. Multi-letter keys like "gg" are vim-style sequences.
var defaultKeys = map[string]string{
	"down": "j down", "up": "k up", "top": "gg home", "bottom": "G end",
	"halfdown": "ctrl+d pgdown", "halfup": "ctrl+u pgup", "current": "o",
	// z x c v b: the classic Winamp prev/play/pause/stop/next row
	"play": "enter", "resume": "x", "toggle": "space c", "stop": "s v", "next": "n b", "prev": "p z",
	"seek-forward": "l right", "seek-back": "h left", "volume-up": "+ =", "volume-down": "-",
	"remove": "dd", "quit": "q", "quit-player": "Q",
	"shuffle": "S", "repeat": "r", "add-files": "a", "add-folder": "A",
	"search": "/ ctrl+j", "help": "?",
}

// parseKeys reads "action = key key" lines and returns key -> action, user bindings winning over defaults.
func parseKeys(conf string) (map[string]string, error) {
	user := map[string]string{}
	for i, line := range strings.Split(conf, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		action, keys, ok := strings.Cut(line, "=")
		action = strings.TrimSpace(action)
		if _, known := defaultKeys[action]; !ok || !known {
			return nil, fmt.Errorf("keys.conf line %d: want `action = keys` with a known action, got %q", i+1, line)
		}
		user[action] = keys
	}
	out := map[string]string{}
	for a, ks := range defaultKeys {
		if _, overridden := user[a]; !overridden {
			for _, k := range strings.Fields(ks) {
				out[k] = a
			}
		}
	}
	for a, ks := range user {
		for _, k := range strings.Fields(ks) {
			out[k] = a
		}
	}
	return out, nil
}

func loadKeys() (map[string]string, error) {
	b, err := os.ReadFile(filepath.Join(configDir(), "keys.conf"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return parseKeys(string(b))
}

// resolve feeds one key press through pending vim-style sequences and returns the action, or "".
func resolve(keys map[string]string, pending *string, k string) string {
	seq := *pending + k
	*pending = ""
	if a, ok := keys[seq]; ok {
		return a
	}
	if len(k) == 1 {
		for b := range keys {
			if len(b) > len(seq) && strings.HasPrefix(b, seq) {
				*pending = seq
				return ""
			}
		}
	}
	if seq != k { // broken sequence: retry the key on its own
		return resolve(keys, pending, k)
	}
	return ""
}
