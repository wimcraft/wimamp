package main

import (
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// match returns playlist indices for q: name or folder contains q first, then fuzzy (in-order letters) matches.
func match(list []entry, q string) []int {
	q = strings.ToLower(q)
	var exact, fuzzy []int
	for i, e := range list {
		name := strings.ToLower(e.name())
		switch {
		case strings.Contains(name, q) || strings.Contains(strings.ToLower(filepath.Dir(e.Filename)), q):
			exact = append(exact, i)
		case inOrder(name, q):
			fuzzy = append(fuzzy, i)
		}
	}
	return append(exact, fuzzy...)
}

func inOrder(s, q string) bool {
	for _, r := range q {
		i := strings.IndexRune(s, r)
		if i < 0 {
			return false
		}
		s = s[i+utf8.RuneLen(r):]
	}
	return true
}

// search is the jump-to-track prompt shared by the TUI and the window.
type search struct {
	on  bool
	q   string
	sel int // index into match(list, q)
}

// key feeds one key (keys.conf naming) to the open prompt and returns the playlist index to play, or -1.
func (s *search) key(k string, list []entry) int {
	switch k {
	case "esc", "ctrl+c":
		s.on = false
	case "enter":
		s.on = false
		if m := match(list, s.q); s.sel < len(m) {
			return m[s.sel]
		}
	case "backspace":
		if rs := []rune(s.q); len(rs) > 0 {
			s.q, s.sel = string(rs[:len(rs)-1]), 0
		}
	case "down", "ctrl+n", "tab":
		s.sel++
	case "up", "ctrl+p":
		s.sel--
	case "space":
		s.q, s.sel = s.q+" ", 0
	default:
		if utf8.RuneCountInString(k) == 1 {
			s.q, s.sel = s.q+k, 0
		}
	}
	s.sel = max(min(s.sel, len(match(list, s.q))-1), 0)
	return -1
}
