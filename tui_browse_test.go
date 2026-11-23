package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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
	items, err := listDir(root)
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
	if _, err := listDir(filepath.Join(root, "missing")); err == nil {
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
