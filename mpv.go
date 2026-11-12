package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func ping() bool {
	c, err := dial()
	if err == nil {
		c.Close()
	}
	return err == nil
}

// ensureDaemon starts a headless mpv listening on our socket unless one already is.
// A lock serializes concurrent starters so there is never more than one mpv.
func ensureDaemon() error {
	if err := endpointErr(); err != nil {
		return err
	}
	if ping() {
		return nil
	}
	unlock, err := lock("daemon", true)
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	defer unlock()
	if ping() { // another wimamp started it while we waited
		return nil
	}
	removeStale()
	cmd := exec.Command(mpvPath(), "--idle=yes", "--no-video", "--no-terminal", "--force-window=no",
		"--input-ipc-server="+socketPath())
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start mpv (is it installed and in PATH?): %w", err)
	}
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if ping() {
			cmd.Process.Release()
			return nil
		}
	}
	cmd.Process.Kill()
	return errors.New("mpv started but did not answer on " + socketPath())
}

// call sends commands over one connection and returns each reply's data (nil when that command failed).
// The returned slice is nil when mpv could not be reached or stopped answering mid-call.
func call(cmds ...[]any) ([]json.RawMessage, error) {
	c, err := dial()
	if err != nil {
		return nil, fmt.Errorf("player not running: %w", err)
	}
	defer c.Close()
	if d, ok := c.(interface{ SetDeadline(time.Time) error }); ok {
		d.SetDeadline(time.Now().Add(10 * time.Second))
	}
	// write from a goroutine: mpv replies while we write, and a big batch would deadlock otherwise
	go func() {
		w := bufio.NewWriter(c)
		enc := json.NewEncoder(w)
		for i, cmd := range cmds {
			enc.Encode(map[string]any{"command": cmd, "request_id": i})
		}
		w.Flush()
	}()

	out := make([]json.RawMessage, len(cmds))
	var first error
	sc := bufio.NewScanner(c)
	sc.Buffer(nil, 64<<20) // a big playlist arrives as one line
	n := 0
	for n < len(cmds) && sc.Scan() {
		var r struct {
			Event     string
			Error     string
			Data      json.RawMessage
			RequestID int `json:"request_id"`
		}
		if json.Unmarshal(sc.Bytes(), &r) != nil || r.Event != "" || r.RequestID < 0 || r.RequestID >= len(cmds) {
			continue
		}
		n++
		if r.Error != "success" {
			if first == nil {
				first = fmt.Errorf("%v: %s", cmds[r.RequestID], r.Error)
			}
			continue
		}
		out[r.RequestID] = r.Data
	}
	if n < len(cmds) {
		return nil, fmt.Errorf("mpv stopped answering: %v", sc.Err())
	}
	return out, first
}

type entry struct {
	Filename, Title string
	Current         bool
}

func (e entry) name() string {
	if e.Title != "" {
		return e.Title
	}
	b := filepath.Base(e.Filename)
	return strings.TrimSuffix(b, filepath.Ext(b))
}

type state struct {
	Title         string
	Pos, Dur, Vol float64
	Paused, Idle  bool
	Shuffle       bool
	Repeat        bool
	List          []entry
}

func fetch() (state, error) {
	r, err := call(
		[]any{"get_property", "playlist"},
		[]any{"get_property", "media-title"},
		[]any{"get_property", "time-pos"},
		[]any{"get_property", "duration"},
		[]any{"get_property", "pause"},
		[]any{"get_property", "idle-active"},
		[]any{"get_property", "volume"},
		[]any{"get_property", "loop-playlist"},
		[]any{"get_property", "user-data/wimamp/shuffle"},
	)
	var s state
	if r == nil {
		return s, err
	}
	// missing properties (e.g. time-pos while stopped) just stay zero
	json.Unmarshal(r[0], &s.List)
	json.Unmarshal(r[1], &s.Title)
	json.Unmarshal(r[2], &s.Pos)
	json.Unmarshal(r[3], &s.Dur)
	json.Unmarshal(r[4], &s.Paused)
	json.Unmarshal(r[5], &s.Idle)
	json.Unmarshal(r[6], &s.Vol)
	s.Repeat = string(r[7]) != "false" && r[7] != nil // "inf", a count, or false
	json.Unmarshal(r[8], &s.Shuffle)
	return s, nil
}

func (s state) line() string {
	if s.Idle {
		return "■ stopped"
	}
	icon := "▶"
	if s.Paused {
		icon = "⏸"
	}
	return fmt.Sprintf("%s %s  %s / %s  vol %d%%", icon, s.Title, fmtTime(s.Pos), fmtTime(s.Dur), int(s.Vol))
}

func fmtTime(sec float64) string {
	t := int(sec)
	return fmt.Sprintf("%d:%02d", t/60, t%60)
}

// resume plays from the current entry, also after `stop`, which leaves mpv idle.
func resume() error {
	r, err := call([]any{"get_property", "idle-active"}, []any{"get_property", "user-data/wimamp/pos"},
		[]any{"get_property", "playlist-count"})
	if r == nil {
		return err
	}
	if string(r[0]) == "true" {
		pos, n := 0, 0
		json.Unmarshal(r[1], &pos)
		json.Unmarshal(r[2], &n)
		if pos < 0 || pos >= n { // the playlist changed since stop
			pos = 0
		}
		_, err = call([]any{"playlist-play-index", pos}, []any{"set_property", "pause", false})
		return err
	}
	_, err = call([]any{"set_property", "pause", false})
	return err
}

// stop keeps the playlist and remembers the position, which mpv itself forgets when it goes idle.
func stop() error {
	r, err := call([]any{"get_property", "playlist-pos"})
	if r == nil {
		return err
	}
	pos := -1
	json.Unmarshal(r[0], &pos)
	if pos < 0 { // already stopped: keep the position saved by the first stop
		_, err = call([]any{"stop", "keep-playlist"})
		return err
	}
	_, err = call([]any{"set_property", "user-data/wimamp/pos", pos}, []any{"stop", "keep-playlist"})
	return err
}

// toggleShuffle remembers the state in mpv, because mpv's playlist-shuffle is a one-shot reorder.
func toggleShuffle() error {
	r, err := call([]any{"get_property", "user-data/wimamp/shuffle"})
	if r == nil {
		return err
	}
	on := string(r[0]) == "true"
	c := "playlist-shuffle"
	if on {
		c = "playlist-unshuffle"
	}
	_, err = call([]any{c}, []any{"set_property", "user-data/wimamp/shuffle", !on})
	return err
}

// playIndex plays entry i; mpv keeps pause across track changes, so unpause too.
func playIndex(i int) error {
	_, err := call([]any{"playlist-play-index", i}, []any{"set_property", "pause", false})
	return err
}

// queue runs player calls one at a time in submission order, so a fast stop+resume or a poll
// racing an action can't land out of order.
var queue = make(chan func(), 256)

func init() {
	go func() {
		for f := range queue {
			f()
		}
	}()
}

func toggleRepeat() error {
	_, err := call([]any{"cycle-values", "loop-playlist", "inf", "no"})
	return err
}

func toggle() error {
	r, err := call([]any{"get_property", "idle-active"})
	if r == nil {
		return err
	}
	if string(r[0]) == "true" {
		return resume()
	}
	_, err = call([]any{"cycle", "pause"})
	return err
}

// simple maps CLI/TUI action names that need no arguments to mpv commands.
var simple = map[string][]any{
	"pause": {"set_property", "pause", true},
	"next":  {"playlist-next"},
	"prev":  {"playlist-prev"},
	"clear": {"stop"}, // plain stop empties the playlist
	"quit":  {"quit"},
}
