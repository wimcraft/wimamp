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

// key feeds one key (keys.conf naming) to the open prompt and returns the playlist index to play
// (or to queue, with ctrl+e, which keeps the prompt open for the next one), or -1.
func (s *search) key(k string, list []entry) (i int, queue bool) {
	switch k {
	case "esc", "ctrl+c":
		s.on = false
	case "enter":
		s.on = false
		if m := match(list, s.q); s.sel < len(m) {
			return m[s.sel], false
		}
	case "ctrl+e": // queue and step to the next match, so a few presses queue several tracks
		if m := match(list, s.q); s.sel < len(m) {
			i := m[s.sel]
			s.sel = min(s.sel+1, len(m)-1)
			return i, true
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
	return -1, false
}

// searchName is how a search result is listed: when q matched a folder rather than the name,
// that folder goes in front, so it is clear why the row matched ("Bad Religion / 03. You Are").
func searchName(e entry, q string) string {
	name := e.name()
	q = strings.ToLower(q)
	if q == "" || strings.Contains(strings.ToLower(name), q) {
		return name
	}
	for d := filepath.Dir(e.Filename); d != filepath.Dir(d); d = filepath.Dir(d) {
		if b := filepath.Base(d); strings.Contains(strings.ToLower(b), q) {
			return b + " / " + name
		}
	}
	return name // a fuzzy match
}
