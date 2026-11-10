package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// version is set at release build time: -ldflags "-X main.version=v0.1.0".
var version = "dev"

const usage = `wimamp — keyboard-first music player on top of mpv

usage:
  wimamp | wimamp tui        open the TUI
  wimamp gui                 open the popup window (bind this to a hotkey)
  wimamp play [paths...]     resume, or replace the playlist with paths and play
  wimamp add <paths...>      append files or folders to the playlist
  wimamp pause | toggle | stop | next | prev
  wimamp seek <[+-]seconds>  wimamp vol <[+-]n>
  wimamp clear               empty the playlist
  wimamp status              print the current track (for status bars)
  wimamp save <name>         save the playlist   wimamp load <name>   load one
  wimamp playlists           list saved playlists
  wimamp keys                list the keys of the TUI and the window
  wimamp quit                stop the background player
  wimamp version             print the version

Keys and playlists live in ` + "`wimamp configdir`" + `.
`

func main() {
	var err error
	if len(os.Args) < 2 {
		err = runTUI()
	} else {
		err = runCmd(os.Args[1], os.Args[2:])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "wimamp:", err)
		os.Exit(1)
	}
}

func runCmd(cmd string, args []string) error {
	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "gui":
		return runGUI()
	case "tui":
		return runTUI()
	case "keys":
		keys, err := loadKeys()
		for _, l := range helpLines(keys) {
			fmt.Printf("%-22s %s\n", l[0], l[1])
		}
		return err
	case "version", "-v", "--version":
		fmt.Println("wimamp", version)
		return nil
	case "configdir":
		fmt.Println(configDir())
		return nil
	case "playlists":
		names, _ := filepath.Glob(filepath.Join(configDir(), "playlists", "*.m3u"))
		for _, n := range names {
			fmt.Println(strings.TrimSuffix(filepath.Base(n), ".m3u"))
		}
		return nil
	case "quit":
		if c, err := dialGUI(); err == nil { // a resident window goes too
			c.Write([]byte("quit\n"))
			c.Close()
		}
		if c, err := dial(); err != nil {
			return nil // nothing running
		} else {
			c.Close()
		}
	}

	if _, ok := simple[cmd]; !ok && !needsPlayer[cmd] { // reject before starting mpv for nothing
		return fmt.Errorf("unknown command %q (see wimamp help)", cmd)
	}
	if err := ensureDaemon(); err != nil {
		return err
	}
	if c, ok := simple[cmd]; ok {
		_, err := call(c)
		return err
	}
	switch cmd {
	case "play":
		if len(args) == 0 {
			return resume()
		}
		return load(args, "replace")
	case "add":
		if len(args) == 0 {
			return errors.New("add: give files or folders")
		}
		return load(args, "append")
	case "toggle":
		return toggle()
	case "stop":
		return stop()
	case "seek", "vol":
		if len(args) != 1 {
			return fmt.Errorf("%s: give one number", cmd)
		}
		c := []any{"seek", args[0]}
		if cmd == "vol" {
			c = []any{"add", "volume", args[0]}
		}
		_, err := call(c)
		return err
	case "status":
		s, err := fetch()
		if err == nil {
			fmt.Println(s.line())
		}
		return err
	case "save":
		if len(args) != 1 {
			return errors.New("save: give a name")
		}
		return save(playlistPath(args[0]))
	case "load":
		if len(args) != 1 {
			return errors.New("load: give a name")
		}
		p := playlistPath(args[0])
		if _, err := os.Stat(p); err != nil {
			return err
		}
		_, err := call([]any{"loadlist", p, "replace"}, []any{"set_property", "pause", false})
		return err
	}
	return fmt.Errorf("unknown command %q (see wimamp help)", cmd)
}

var needsPlayer = map[string]bool{
	"play": true, "add": true, "toggle": true, "stop": true, "seek": true, "vol": true,
	"status": true, "save": true, "load": true,
}

// load adds files/folders to the playlist; mode "replace" also starts playback.
func load(args []string, mode string) error {
	files, err := collect(args)
	if err != nil {
		return err
	}
	cmds := make([][]any, 0, len(files)+1)
	for i, f := range files {
		m := "append"
		if i == 0 {
			m = mode
		}
		cmds = append(cmds, []any{"loadfile", f, m})
	}
	if mode == "replace" {
		cmds = append(cmds, []any{"set_property", "pause", false})
	}
	_, err = call(cmds...)
	return err
}

var audioExt = map[string]bool{
	".mp3": true, ".flac": true, ".ogg": true, ".oga": true, ".opus": true, ".m4a": true, ".mp4": true,
	".aac": true, ".wav": true, ".aif": true, ".aiff": true, ".wma": true, ".ape": true, ".wv": true,
	".mka": true, ".webm": true, ".alac": true,
}

// collect turns args into absolute file paths (mpv runs elsewhere), walking folders for audio files.
func collect(args []string) ([]string, error) {
	var out []string
	for _, a := range args {
		if strings.Contains(a, "://") {
			out = append(out, a)
			continue
		}
		abs, err := filepath.Abs(a)
		if err != nil {
			return nil, err
		}
		fi, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			out = append(out, abs)
			continue
		}
		filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && audioExt[strings.ToLower(filepath.Ext(p))] {
				out = append(out, p)
			}
			return nil // skip unreadable entries, keep walking
		})
	}
	if len(out) == 0 {
		return nil, errors.New("no audio files found")
	}
	return out, nil
}

func save(path string) error {
	r, err := call([]any{"get_property", "playlist"})
	if err != nil {
		return err
	}
	var list []entry
	json.Unmarshal(r[0], &list)
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for _, e := range list {
		b.WriteString(e.Filename + "\n")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// playlistPath maps a bare name to the playlists dir; anything path-like is used as given.
func playlistPath(name string) string {
	if strings.ContainsAny(name, `/\`) || strings.HasSuffix(name, ".m3u") {
		p, _ := filepath.Abs(name)
		return p
	}
	return filepath.Join(configDir(), "playlists", name+".m3u")
}

func configDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "wimamp")
	}
	if runtime.GOOS == "windows" {
		d, _ := os.UserConfigDir()
		return filepath.Join(d, "wimamp")
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".config", "wimamp")
}
