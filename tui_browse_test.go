package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// musicTree makes root/{b.mp3, A.flac, cover.jpg, .hidden.mp3, cd2/{x.ogg}, Zed/, link->cd2}.
func musicTree(t *testing.T) string {
	root := t.TempDir()
	for _, f := range []string{"b.mp3", "A.flac", "cover.jpg", ".hidden.mp3", "cd2/x.ogg", "Zed/.keep"} {
		p := filepath.Join(root, f)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, nil, 0o644)
	}
	os.Symlink(filepath.Join(root, "cd2"), filepath.Join(root, "link"))
	return root
}

func TestListDir(t *testing.T) {
	root := musicTree(t)
	items, err := listDir(root, audioExt)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, it := range items {
		got = append(got, it.name+map[bool]string{true: "/"}[it.dir])
	}
	if want := []string{"cd2/", "link/", "Zed/", "A.flac", "b.mp3"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if _, err := listDir(filepath.Join(root, "missing"), audioExt); err == nil {
		t.Error("no error for a missing folder")
	}
}

func TestBrowserKeys(t *testing.T) {
	root := musicTree(t)
	m := testModel(50, 14)
	m.keys, _ = parseKeys("")
	m.s.BrowseDir = root
	m.do("add-files")
	if !m.br.on || m.br.dir != root {
		t.Fatalf("browser on %v in %q", m.br.on, m.br.dir)
	}
	press := func(keys ...string) {
		for _, k := range keys {
			m.key(k)
		}
	}
	press("enter") // into cd2/
	if m.br.dir != filepath.Join(root, "cd2") {
		t.Fatalf("enter on a folder: in %q", m.br.dir)
	}
	press("space", "h") // mark x.ogg, go up: cd2/ selected again
	if m.br.dir != root || m.br.sel != 0 {
		t.Fatalf("up: in %q sel %d", m.br.dir, m.br.sel)
	}
	press("G", "space", "k", "k", "space") // mark b.mp3, then Zed/
	want := []string{filepath.Join(root, "cd2", "x.ogg"), filepath.Join(root, "b.mp3"), filepath.Join(root, "Zed")}
	if got := m.br.take(); !slices.Equal(got, want) {
		t.Errorf("marked %v, want %v", got, want)
	}
	if !strings.Contains(strings.Join(screen(m), "\n"), "3 marked") {
		t.Error("marked count not shown")
	}
	checkSize(t, screen(m), 50, 14)
	press("k", "space") // Zed/ unmarked again
	if len(m.br.take()) != 2 {
		t.Errorf("unmark: %v", m.br.take())
	}
	press("x") // a player key: ignored while browsing
	if !m.br.on {
		t.Fatal("x closed the browser")
	}
	if m.key("esc") == nil || m.br.on || m.s.BrowseDir != root {
		t.Errorf("esc: on %v, kept dir %q", m.br.on, m.s.BrowseDir)
	}
	m.br.marks = nil
	m.do("add-folder") // reopens where it was left
	press("j", "j", "j") // A.flac
	if got := m.br.take(); len(got) != 1 || filepath.Base(got[0]) != "A.flac" {
		t.Errorf("selected file: %v", got)
	}
}

func TestBrowserView(t *testing.T) {
	m := testModel(30, 12)
	m.br = browser{on: true, dir: "/very/long/path/to/some/music/folder", items: []browseItem{{"CD1", true}, {"01 Intro.mp3", false}}}
	lines := screen(m)
	checkSize(t, lines, 30, 12)
	if !strings.Contains(lines[5], "…") || !strings.Contains(lines[5], "folder") || !strings.Contains(lines[6], "CD1/") {
		t.Errorf("title or rows:\n%s", strings.Join(lines, "\n"))
	}
}

func TestBrowserSearch(t *testing.T) {
	root := musicTree(t) // cd2/ link/ Zed/ A.flac b.mp3
	m := testModel(50, 14)
	m.keys, _ = parseKeys("")
	m.s.BrowseDir = root
	m.do("add-files")
	press := func(keys ...string) {
		for _, k := range keys {
			m.key(k)
		}
	}
	names := func() (out []string) {
		for _, it := range m.br.items {
			out = append(out, it.name)
		}
		return out
	}
	press("/", "Z") // case ignored: Zed/
	if got := names(); !slices.Equal(got, []string{"Zed"}) {
		t.Errorf("/Z: %v", got)
	}
	if !strings.Contains(strings.Join(screen(m), "\n"), "/Z█  1 matches") {
		t.Error("search prompt not shown")
	}
	press("backspace", "d", "2") // cd2 only; j/k-style letters are text while typing
	if got := names(); !slices.Equal(got, []string{"cd2"}) || !m.br.typing {
		t.Errorf("/d2: %v typing %v", got, m.br.typing)
	}
	press("enter") // opens the match and clears the search
	if m.br.dir != filepath.Join(root, "cd2") || m.br.q != "" || m.br.typing {
		t.Fatalf("enter: in %q, q %q", m.br.dir, m.br.q)
	}
	press("h", "/", "m", "p", "tab", "esc") // mark b.mp3 from the search, then clear it
	if got := m.br.take(); len(got) != 1 || filepath.Base(got[0]) != "b.mp3" || m.br.q != "" || len(m.br.items) != 5 {
		t.Errorf("tab in search: marks %v, q %q, %d items", got, m.br.q, len(m.br.items))
	}
	press("/", "f", "l", "a", "c", "enter") // enter on a file adds it and closes
	if m.br.on {
		t.Error("enter on a matched file did not add")
	}
	m.do("add-files")
	press("/", "q", "q", "enter", "esc") // nothing matches; enter keeps the search, esc clears it
	if !m.br.on || m.br.q != "" || len(m.br.items) != 5 {
		t.Errorf("after esc: on %v q %q items %d", m.br.on, m.br.q, len(m.br.items))
	}
	press("esc")
	if m.br.on {
		t.Error("second esc did not close")
	}
	checkSize(t, screen(m), 50, 14)
}

func TestTUIPlaylistKeys(t *testing.T) {
	list := startTestPlayer(t, 3, 5)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := testModel(50, 14)
	m.keys, _ = parseKeys("")
	exec := func(c tea.Cmd) {
		if c != nil {
			if msg, ok := c().(flashMsg); ok {
				t.Fatalf("error: %s", msg)
			}
		}
	}
	key := func(keys ...string) {
		for _, k := range keys {
			exec(m.key(k))
		}
	}
	key("ctrl+s", "e", "v", "x", "backspace")
	if !strings.Contains(strings.Join(screen(m), "\n"), "save playlist as: ev█") {
		t.Errorf("save prompt:\n%s", strings.Join(screen(m), "\n"))
	}
	key("e", "enter") // saved as <config>/playlists/eve.m3u
	if m.saving {
		t.Fatal("enter did not save")
	}
	os.WriteFile(filepath.Join(configDir(), "playlists", "notes.txt"), nil, 0o644)
	key("ctrl+n")
	if s, _ := fetch(); len(s.List) != 0 {
		t.Fatalf("ctrl+n left %d entries", len(s.List))
	}
	key("ctrl+o")
	if !m.br.on || !m.br.lists || len(m.br.items) != 1 || m.br.items[0].name != "eve.m3u" {
		t.Fatalf("ctrl+o lists %+v", m.br.items)
	}
	if !strings.Contains(strings.Join(screen(m), "\n"), "LOAD ") {
		t.Error("load title not shown")
	}
	key("space") // no marks when loading
	if len(m.br.marks) != 0 {
		t.Error("marked a playlist")
	}
	key("enter")
	time.Sleep(200 * time.Millisecond)
	s, _ := fetch()
	if m.br.on || len(s.List) != 3 || s.List[0].Filename != list[0].Filename {
		t.Errorf("after load: browser %v, %d entries", m.br.on, len(s.List))
	}
	key("ctrl+s", "esc") // cancel
	if m.saving {
		t.Error("esc did not cancel the save prompt")
	}
}

func TestQueueViewKeys(t *testing.T) {
	m := testModel(50, 14)
	m.keys, _ = parseKeys("")
	m.queueOn = true
	if m.key("e") != nil || !m.queueOn {
		t.Error("e acted in the queue view")
	}
	if m.key("d") == nil || m.pending != "" {
		t.Error("a single d did not unqueue")
	}
}
