package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// The TUI is drawn as two boxes in the window's palette: the player above, the playlist below.

const playerLines = 5 // the player box, borders included

var (
	stBorder = fg(cBtn)
	stTitle  = fg(cWhite).Bold(true)
	stText   = fg(cGreen)
	stPlay   = fg(cWhite).Bold(true)
	stDim    = fg(cDim)
	stGold   = fg(cGold)
	stErr    = fg(cErr)
	selBg    = lipgloss.Color(hex(cSel))
)

func hex(c color.RGBA) string        { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }
func fg(c color.RGBA) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(hex(c))) }
func cols(s string) int              { return ansi.StringWidth(s) } // terminal columns, wide characters counted as two
func cut(s string, w int) string     { return ansi.Truncate(s, max(w, 0), "…") }
func cutLeft(s string, n int) string { return ansi.TruncateLeft(s, n, "") } // drops n columns from the left
func border(s string) string         { return stBorder.Render(s) }
func rule(n int) string              { return strings.Repeat("─", max(n, 0)) }

func padTo(s string, w int) string {
	s = cut(s, w)
	return s + strings.Repeat(" ", max(0, w-cols(s)))
}

// spread puts left and right at the ends of a w-column line, cutting left when they don't fit.
func spread(left, right string, w int) string {
	if right == "" {
		return padTo(left, w)
	}
	left = cut(left, w-cols(right)-1)
	return left + strings.Repeat(" ", max(1, w-cols(left)-cols(right))) + right
}

// boxTop is "╭── title ──── right ╮", or the title centered when there is no right part.
func boxTop(title, right string, w int) string {
	t := " " + title + " "
	if right != "" {
		r := " " + right + " "
		if n := w - 4 - cols(t) - cols(r); n >= 1 {
			return border("╭──") + stTitle.Render(t) + border(rule(n)) + stText.Render(r) + border("╮")
		}
	}
	n := w - 2 - cols(t)
	return border("╭"+rule(n/2)) + stTitle.Render(t) + border(rule(n-n/2)+"╮")
}

func boxBottom(text string, w int) string {
	if text == "" {
		return border("╰" + rule(w-2) + "╯")
	}
	text = cut(text, w-6)
	return border("╰─ ") + text + border(" "+rule(w-5-cols(text))+"╯")
}

func boxRow(s string, w int) string { return border("│") + " " + padTo(s, w-4) + " " + border("│") }

func (m *model) View() string {
	w := m.w
	if w == 0 {
		w = 80 // before the first size message
	}
	w = max(w, 24)
	return strings.Join(append(m.playerBox(w), m.listBox(w)...), "\n")
}

func (m *model) playerBox(w int) []string {
	in, s := w-4, m.s
	var top string
	switch {
	case m.err != nil:
		top = stErr.Render("⚠ " + m.err.Error())
	case s.Idle:
		top = stGold.Render("■") + "  " + stDim.Render("stopped")
	default:
		icon := "▶"
		if s.Paused {
			icon = "❚❚"
		}
		title := s.Title
		if i := currentIndex(s.List, -1); i >= 0 {
			title = fmt.Sprintf("%d. %s", i+1, title)
		}
		top = stGold.Render(icon) + "  " + stPlay.Render(fmtTime(s.Pos)+" / "+fmtTime(s.Dur)) + "   " + stText.Render(title)
	}
	// seek bar: elapsed in gold, a knob, the rest dim
	seek := stDim.Render(rule(in))
	if m.err == nil && !s.Idle && s.Dur > 0 {
		k := max(0, min(int(s.Pos/s.Dur*float64(in-1)), in-1))
		seek = stGold.Render(strings.Repeat("━", k)) + stPlay.Render("●") + stDim.Render(rule(in-1-k))
	}
	// volume meter, colored by level like the window's slider; it shrinks on narrow terminals
	cells := min(10, max(in-26, 3))
	n := min(cells, int(s.Vol/100*float64(cells)+0.5))
	vol := "vol " + fg(volColor(s.Vol)).Render(strings.Repeat("▮", n)) + stDim.Render(strings.Repeat("▯", cells-n)) +
		stText.Render(fmt.Sprintf(" %d%%", int(s.Vol)))
	light := func(on bool, name string) string {
		if on {
			return stText.Render("● " + name)
		}
		return stDim.Render("○ " + name)
	}
	return []string{
		boxTop("WIMAMP", "", w),
		boxRow(top, w),
		boxRow(seek, w),
		boxRow(spread(vol, light(s.Shuffle, "shuf")+"  "+light(s.Repeat, "rep"), in), w),
		boxBottom("", w),
	}
}

// listBox is the playlist box: m.h rows between its borders.
func (m *model) listBox(w int) []string {
	if m.br.on && !m.help {
		return m.browseBox(w)
	}
	if m.info != nil && !m.help {
		return m.infoBox(w)
	}
	in, s, h := w-4, m.s, m.h
	// rows to show: the whole playlist, the matches while searching, or the queue
	idx, sel, top := make([]int, len(s.List)), m.cursor, m.top
	for i := range idx {
		idx[i] = i
	}
	title := "PLAYLIST"
	right := strings.TrimSpace(fmt.Sprintf("%d/%d  %s", min(m.cursor+1, len(s.List)), len(s.List), totalLength(s.List)))
	hint := stDim.Render("/ search  e queue  E queue view  ? keys")
	var body []string // text lines above the rows
	switch {
	case m.help:
		for _, l := range helpLines(m.keys) {
			body = append(body, stText.Render(fmt.Sprintf("%-22s %s", l[0], l[1])))
		}
		title, right, hint, idx = "KEYS", "", stDim.Render("press any key"), nil
	case m.srch.on:
		idx, sel = match(s.List, m.srch.q), m.srch.sel
		top = max(0, sel-h+1)
		hint = stPlay.Render(fmt.Sprintf("/%s█  %d matches", m.srch.q, len(idx)))
	case m.queueOn:
		idx, sel = queueIndices(s.List, s.Queue), m.qsel
		top = max(0, sel-h+1)
		title, right = fmt.Sprintf("QUEUE (%d)", len(idx)), ""
		hint = stDim.Render("e/dd unqueue  J/K move  esc back")
		if len(idx) == 0 {
			body = append(body, stDim.Render("queue is empty — select a track and press e"))
		}
	case len(s.List) == 0:
		body = append(body, stDim.Render("playlist is empty — add music with: wimamp add ~/Music"))
	}
	if m.flash != "" {
		hint = stErr.Render(m.flash)
	}
	lines := []string{boxTop(title, right, w)}
	for _, l := range body {
		lines = append(lines, boxRow(l, w))
	}
	var visible []entry
	for n := top; n < len(idx) && n < top+h; n++ {
		e := s.List[idx[n]]
		visible = append(visible, e)
		mark, st := "  ", stText
		if e.Current { // as in Winamp: the playing entry is white
			mark, st = "▶ ", stPlay
		}
		row := spread(fmt.Sprintf("%s%d. %s%s", mark, idx[n]+1, queueMark(e, s.Queue), m.rowName(e)), rowLength(e, s), in)
		if n == sel { // the selection bar spans the box
			lines = append(lines, border("│")+st.Background(selBg).Render(" "+row+" ")+border("│"))
			continue
		}
		lines = append(lines, boxRow(st.Render(row), w))
	}
	queueMissing(visible, true)
	for len(lines) < h+1 {
		lines = append(lines, boxRow("", w))
	}
	return append(lines[:h+1], boxBottom(hint, w))
}

// infoBox is the track info view in the playlist box: label/value rows, long values wrapped.
func (m *model) infoBox(w int) []string {
	const lw = 13 // label column
	in, h := w-4, m.h
	lines := []string{boxTop("TRACK INFO", "", w)}
	add := func(s string) {
		if len(lines) <= h {
			lines = append(lines, boxRow(s, w))
		}
	}
	if m.info.rows == nil {
		add(stDim.Render(m.info.e.name() + " — reading…"))
	}
	for _, r := range m.info.rows {
		label := r[0]
		for _, part := range strings.Split(ansi.Hardwrap(r[1], max(in-lw, 4), true), "\n") {
			add(stDim.Render(padTo(label, lw)) + stText.Render(part))
			label = ""
		}
	}
	for len(lines) < h+1 {
		lines = append(lines, boxRow("", w))
	}
	hint := stDim.Render("f show in folder  esc back")
	if m.flash != "" {
		hint = stErr.Render(m.flash)
	}
	return append(lines, boxBottom(hint, w))
}
