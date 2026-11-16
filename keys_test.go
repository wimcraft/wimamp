package main

import (
	"strings"
	"testing"
)

func TestKeys(t *testing.T) {
	keys, err := parseKeys("# mine\nnext = n J\ndown = down\n")
	if err != nil {
		t.Fatal(err)
	}
	if keys["J"] != "next" || keys["j"] != "" || keys["down"] != "down" {
		t.Fatalf("override failed: %v", keys)
	}
	if _, err := parseKeys("# c\nbogus = x"); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("unknown action: want error naming line 2, got %v", err)
	}
	unbound, _ := parseKeys("next =")
	if unbound["n"] != "" || unbound["b"] != "" {
		t.Fatal("empty binding should unbind next")
	}
	defaults, _ := parseKeys("")
	for k, a := range map[string]string{"z": "prev", "x": "resume", "c": "toggle", "v": "stop", "b": "next", "o": "current", "space": "toggle", "pgdown": "halfdown"} {
		if defaults[k] != a {
			t.Errorf("default %s = %q, want %q", k, defaults[k], a)
		}
	}

	var p string
	press := func(k string) string { return resolve(keys, &p, k) }
	if a := press("g"); a != "" || p != "g" {
		t.Fatalf("g should wait, got %q pending %q", a, p)
	}
	if a := press("g"); a != "top" {
		t.Fatalf("gg = %q", a)
	}
	press("g")
	if a := press("G"); a != "bottom" { // broken sequence falls back to the lone key
		t.Fatalf("gG = %q", a)
	}
	press("d")
	if a := press("d"); a != "remove" {
		t.Fatalf("dd = %q", a)
	}
	if a := press("enter"); a != "play" {
		t.Fatalf("enter = %q", a)
	}
}

func TestSearch(t *testing.T) {
	list := []entry{
		{Filename: "/m/Each Coffee Hour.mp3"},
		{Filename: "/m/Echo.mp3"},
		{Filename: "/m/Night Ferry/Static Bloom/Stop!.m4a"},
		{Filename: "/m/x.mp3", Title: "Night Ferry - Echo"},
	}
	got := match(list, "ECHO")
	if len(got) != 3 || got[0] != 1 || got[1] != 3 || got[2] != 0 {
		t.Fatalf("echo: want [1 3 0] (contains first, then fuzzy), got %v", got)
	}
	if got := match(list, "static bloom"); len(got) != 1 || got[0] != 2 {
		t.Fatalf("folder match: %v", got)
	}

	s := search{on: true}
	for _, k := range []string{"e", "c", "h", "o", "down"} {
		s.key(k, list)
	}
	if s.sel != 1 {
		t.Fatalf("sel = %d", s.sel)
	}
	if idx, q := s.key("enter", list); idx != 3 || q || s.on {
		t.Fatalf("enter: idx %d queue %v on %v", idx, q, s.on)
	}
	s = search{on: true}
	for _, k := range []string{"e", "c", "h", "o", "down"} {
		s.key(k, list)
	}
	if idx, q := s.key("ctrl+e", list); idx != 3 || !q || !s.on || s.q != "echo" || s.sel != 2 {
		t.Fatalf("ctrl+e: idx %d queue %v on %v query %q sel %d", idx, q, s.on, s.q, s.sel)
	}
	if idx, _ := s.key("ctrl+e", list); idx != 0 || s.sel != 2 { // the last match stays selected
		t.Fatalf("second ctrl+e: idx %d sel %d", idx, s.sel)
	}
	s = search{on: true}
	s.key("x", list)
	s.key("backspace", list)
	if i, _ := s.key("esc", list); s.q != "" || i != -1 || s.on {
		t.Fatal("backspace/esc")
	}
}

func TestHelpReflectsOverrides(t *testing.T) {
	keys, _ := parseKeys("next = J")
	for _, l := range helpLines(keys) {
		if l[0] == "next / prev track" && l[1] != "J / p z" {
			t.Fatalf("got %q", l[1])
		}
	}
}

func TestSearchName(t *testing.T) {
	e := entry{Filename: "/music/Bad Religion/1988 - Suffer/03. You Are.mp3"}
	for q, want := range map[string]string{
		"religion": "Bad Religion / 03. You Are", // matched the artist folder
		"suffer":   "1988 - Suffer / 03. You Are",
		"you are":  "03. You Are", // matched the name: no folder needed
		"":         "03. You Are",
		"brlgn":    "03. You Are", // fuzzy
	} {
		if got := searchName(e, q); got != want {
			t.Errorf("%q: %q, want %q", q, got, want)
		}
	}
}
