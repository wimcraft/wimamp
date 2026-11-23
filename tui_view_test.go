package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func testModel(w, h int) *model {
	m := &model{w: w, h: h - playerLines - 2, keys: map[string]string{}}
	m.s = state{Title: "The Concept Stays", Pos: 38, Dur: 186, Vol: 50, Paused: true, Shuffle: true}
	for _, n := range []string{"Americans Abroad", "Stop!", "02 - Shattered Faith", "The Concept Stays", "09 - Get Alone"} {
		m.s.List = append(m.s.List, entry{Filename: "/m/" + n + ".mp3", ID: len(m.s.List) + 1})
	}
	m.s.List[3].Current = true
	m.s.Queue = []int{3}
	m.cursor = 4
	return m
}

func screen(m *model) []string { return strings.Split(ansi.Strip(m.View()), "\n") }

func checkSize(t *testing.T, lines []string, w, h int) {
	t.Helper()
	if len(lines) != h {
		t.Errorf("%d lines, want %d", len(lines), h)
	}
	for i, l := range lines {
		if cols(l) != w {
			t.Errorf("line %d is %d columns, want %d: %q", i, cols(l), w, l)
		}
	}
}

func TestTUIView(t *testing.T) {
	resetLengths()
	got := screen(testModel(50, 14))
	want := `╭──────────────────── WIMAMP ────────────────────╮
│ ❚❚  0:38 / 3:06   4. The Concept Stays         │
│ ━━━━━━━━━●──────────────────────────────────── │
│ vol ▮▮▮▮▮▯▯▯▯▯ 50%               ● shuf  ○ rep │
╰────────────────────────────────────────────────╯
╭── PLAYLIST ──────────────────────── 5/5  0:00+ ╮
│   1. Americans Abroad                          │
│   2. Stop!                                     │
│   3. [1] 02 - Shattered Faith                  │
│ ▶ 4. The Concept Stays                    3:06 │
│   5. 09 - Get Alone                            │
│                                                │
│                                                │
╰─ / search  e queue  E queue view  ? keys ──────╯`
	if g := strings.Join(got, "\n"); g != want {
		t.Errorf("got\n%s\nwant\n%s", g, want)
	}
	checkSize(t, got, 50, 14)
}

func TestTUIViewStates(t *testing.T) {
	resetLengths()
	for name, c := range map[string]struct {
		set  func(m *model)
		find string
	}{
		"search": {func(m *model) { m.srch = search{on: true, q: "stop"} }, "/stop█  1 matches"},
		"error":  {func(m *model) { m.flash = "file not found" }, "╰─ file not found"},
		"queue":  {func(m *model) { m.queueOn = true }, "QUEUE (1)"},
		"help":   {func(m *model) { m.help = true }, "press any key"},
		"empty":  {func(m *model) { m.s.List, m.s.Idle = nil, true }, "playlist is empty"},
		"down":   {func(m *model) { m.err = errDown }, "⚠ player not running"},
	} {
		m := testModel(50, 14)
		c.set(m)
		lines := screen(m)
		if !strings.Contains(strings.Join(lines, "\n"), c.find) {
			t.Errorf("%s: no %q in\n%s", name, c.find, strings.Join(lines, "\n"))
		}
		checkSize(t, lines, 50, 14)
	}
}

var errDown = errString("player not running")

type errString string

func (e errString) Error() string { return string(e) }

func TestTUIViewNarrow(t *testing.T) {
	resetLengths()
	for _, w := range []int{24, 30, 41} {
		checkSize(t, screen(testModel(w, 12)), w, 12)
	}
	m := testModel(40, 12)
	m.s.List[0].Filename = "/m/ゴッドスマック - アウェイク ゴッドスマック.mp3" // two columns per character
	checkSize(t, screen(m), 40, 12)
	m = testModel(10, 12) // narrower than the minimum: drawn at the minimum
	checkSize(t, screen(m), 24, 12)
}
