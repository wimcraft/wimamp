package main

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestQueueHelpers(t *testing.T) {
	if got := toggleID([]int{1, 2}, 3); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("queue: %v", got)
	}
	if got := toggleID([]int{1, 2, 3}, 2); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("unqueue by toggling: %v", got)
	}
	if got := unqueueID([]int{1, 2}, 5); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("unqueue missing: %v", got)
	}
	for _, c := range []struct {
		id, d int
		want  []int
	}{{1, 1, []int{2, 1, 3}}, {3, -1, []int{1, 3, 2}}, {1, -1, []int{1, 2, 3}}, {3, 1, []int{1, 2, 3}}, {9, 1, []int{1, 2, 3}}} {
		if got := moveID([]int{1, 2, 3}, c.id, c.d); !slices.Equal(got, c.want) {
			t.Errorf("move %d by %d: %v, want %v", c.id, c.d, got, c.want)
		}
	}
	list := []entry{{ID: 10}, {ID: 11}, {ID: 12}}
	if got := queueIndices(list, []int{12, 99, 10}); !slices.Equal(got, []int{2, 0}) {
		t.Errorf("indices: %v", got)
	}
	if queueMark(list[2], []int{11, 12}) != "[2] " || queueMark(list[0], []int{11}) != "" {
		t.Error("marks")
	}
	if queueAction("remove") != "unqueue" || queueAction("queue") != "unqueue" || queueAction("next") != "next" {
		t.Error("queue view actions")
	}
}

// silentWAV writes a valid mono 16-bit PCM file of the given length.
func silentWAV(t *testing.T, p string, sec float64) {
	const rate = 8000
	n := uint32(sec * rate * 2)
	h := []byte("RIFF\x00\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x01\x00")
	h = binary.LittleEndian.AppendUint32(h, rate)
	h = binary.LittleEndian.AppendUint32(h, rate*2)
	h = append(h, 2, 0, 16, 0)
	h = append(append(h, "data"...), binary.LittleEndian.AppendUint32(nil, n)...)
	binary.LittleEndian.PutUint32(h[4:], uint32(len(h))-8+n)
	if err := os.WriteFile(p, append(h, make([]byte, n)...), 0o644); err != nil {
		t.Fatal(err)
	}
}

// startTestPlayer runs a private mpv with the queue script and n short tracks loaded, paused on the first.
func startTestPlayer(t *testing.T, n int, sec float64) []entry {
	if _, err := exec.LookPath("mpv"); err != nil {
		t.Skip("mpv not installed")
	}
	dir, err := os.MkdirTemp("/tmp", "wq") // short: unix socket paths are limited to ~104 bytes
	if err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(dir, "s")
	t.Setenv("WIMAMP_SOCKET", sock)
	// registered after Setenv so it runs before the variable is restored: quit only this test's player
	t.Cleanup(func() {
		if os.Getenv("WIMAMP_SOCKET") == sock {
			call(simple["quit"])
		}
		time.Sleep(100 * time.Millisecond)
		os.RemoveAll(dir)
	})
	if err := ensureDaemon(); err != nil {
		t.Fatal(err)
	}
	cmds := [][]any{{"set_property", "ao", "null"}, {"set_property", "pause", true}}
	for i := range n {
		p := filepath.Join(dir, string(rune('a'+i))+".wav")
		silentWAV(t, p, sec)
		mode := "append-play"
		if i > 0 {
			mode = "append"
		}
		cmds = append(cmds, []any{"loadfile", p, mode})
	}
	if _, err := call(cmds...); err != nil {
		t.Fatal(err)
	}
	list, _, err := readQueue()
	if err != nil || len(list) != n {
		t.Fatalf("playlist %v %v", list, err)
	}
	return list
}

// playOrder unpauses and records the playlist index of each track that plays until the player idles.
func playOrder(t *testing.T) []int {
	call([]any{"set_property", "pause", false})
	var order []int
	var idleSince time.Time
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		r, _ := call([]any{"get_property", "playlist-playing-pos"}, []any{"get_property", "idle-active"})
		var pos int
		var idle bool
		if r != nil {
			json.Unmarshal(r[0], &pos)
			json.Unmarshal(r[1], &idle)
		}
		switch {
		case !idle:
			idleSince = time.Time{}
		case idleSince.IsZero():
			idleSince = time.Now()
		case len(order) > 0 && time.Since(idleSince) > 500*time.Millisecond: // stayed idle: done
			return order
		}
		if pos >= 0 && (len(order) == 0 || order[len(order)-1] != pos) {
			order = append(order, pos)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("still playing after 10 s: %v", order)
	return nil
}

func TestQueuePlaysFirst(t *testing.T) {
	list := startTestPlayer(t, 4, 0.4)
	if err := setQueue([]int{list[3].ID, list[2].ID}); err != nil {
		t.Fatal(err)
	}
	// after the queue, the playlist goes on after the track that played before it
	if got := playOrder(t); !slices.Equal(got, []int{0, 3, 2, 1, 2, 3}) {
		t.Errorf("played %v, want [0 3 2 1 2 3]", got)
	}
}

func TestQueueAfterLastTrack(t *testing.T) {
	list := startTestPlayer(t, 2, 0.4)
	call([]any{"playlist-play-index", 1}) // stays paused
	for i := 0; i < 100; i++ {
		if r, _ := call([]any{"get_property", "playlist-playing-pos"}); r != nil && string(r[0]) == "1" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	setQueue([]int{list[0].ID})
	// the queued entry plays after the last one ends, then the playlist stops: it was at its end
	if got := playOrder(t); !slices.Equal(got, []int{1, 0}) {
		t.Errorf("played %v, want [1 0]", got)
	}
}

func TestQueueNextAndPick(t *testing.T) {
	list := startTestPlayer(t, 4, 5)
	setQueue([]int{list[2].ID, list[3].ID})
	if err := next(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	s, _ := fetch()
	if i := currentIndex(s.List, -1); i != 2 || !slices.Equal(s.Queue, []int{list[3].ID}) {
		t.Errorf("after next: playing %d, queue %v", i, s.Queue)
	}
	next()
	next() // queue drained: back after the track playing before it
	time.Sleep(200 * time.Millisecond)
	if s, _ := fetch(); currentIndex(s.List, -1) != 1 {
		t.Errorf("after the queue: playing %d, want 1", currentIndex(s.List, -1))
	}
	setQueue([]int{list[3].ID})
	playIndex(3) // picked by hand: leaves the queue
	time.Sleep(200 * time.Millisecond)
	if s, _ := fetch(); len(s.Queue) != 0 {
		t.Errorf("queue after picking the queued entry: %v", s.Queue)
	}
}
