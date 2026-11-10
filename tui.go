package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg struct{}
type stateMsg struct {
	s   state
	err error
}
type flashMsg string

type model struct {
	s              state
	err            error
	flash, pending string
	keys           map[string]string
	cursor, top, h int
	synced, help   bool
	srch           search
}

func runTUI() error {
	keys, err := loadKeys()
	if err != nil {
		return err
	}
	if err := ensureDaemon(); err != nil {
		return err
	}
	_, err = tea.NewProgram(&model{keys: keys, h: 20}, tea.WithAltScreen()).Run()
	return err
}

// refresh fetches state through the player queue, so it can't overtake an earlier action.
func refresh() tea.Msg { return run(nil)() }

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

// run executes a player command off the UI thread, then refreshes.
// run executes a player call (nil just fetches) in queue order, then returns the fresh state.
func run(f func() error) tea.Cmd {
	return func() tea.Msg {
		done := make(chan tea.Msg, 1)
		queue <- func() {
			if f != nil {
				if err := f(); err != nil {
					done <- flashMsg(err.Error())
					return
				}
			}
			s, err := fetch()
			done <- stateMsg{s, err}
		}
		return <-done
	}
}

func (m *model) Init() tea.Cmd { return tea.Batch(refresh, tick()) }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.h = max(msg.Height-2, 1)
	case tickMsg:
		return m, tea.Batch(refresh, tick())
	case flashMsg:
		m.flash = string(msg)
	case stateMsg:
		m.s, m.err = msg.s, msg.err
		if !m.synced && m.err == nil {
			m.cursor = currentIndex(m.s.List, 0)
			m.synced = true
		}
	case tea.KeyMsg:
		k := msg.String()
		if k == " " {
			k = "space"
		}
		if k == "ctrl+c" {
			return m, tea.Quit
		}
		m.flash = ""
		keys := []string{k}
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 1 && !msg.Paste { // fast typing arrives as one msg
			keys = strings.Split(string(msg.Runes), "")
		}
		var cmds []tea.Cmd
		for _, k := range keys {
			cmds = append(cmds, m.key(k))
			m.clamp()
		}
		return m, tea.Sequence(cmds...) // in order: a fast "stop, play" must not run as "play, stop"
	}
	m.clamp()
	return m, nil
}

// key routes one key to the help, the search prompt, or the bindings, whichever is active.
func (m *model) key(k string) tea.Cmd {
	switch {
	case m.help: // any key closes the help
		m.help = false
	case m.srch.on:
		if i := m.srch.key(k, m.s.List); i >= 0 {
			m.cursor = i
			return run(playerAction("play", i))
		}
	default:
		return m.do(resolve(m.keys, &m.pending, k))
	}
	return nil
}

func (m *model) do(a string) tea.Cmd {
	if c, ok := moveCursor(a, m.cursor, m.h, m.s.List); ok {
		m.cursor = c
		return nil
	}
	switch a {
	case "search":
		m.srch = search{on: true}
	case "help":
		m.help = true
	case "quit":
		return tea.Quit
	case "quit-player":
		return tea.Sequence(func() tea.Msg { call(simple["quit"]); return nil }, tea.Quit)
	}
	if f := playerAction(a, m.cursor); f != nil {
		return run(f)
	}
	return nil // e.g. add-files: window only
}

func (m *model) clamp() {
	m.cursor = max(min(m.cursor, len(m.s.List)-1), 0)
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+m.h {
		m.top = m.cursor - m.h + 1
	}
}

func (m *model) View() string {
	var b strings.Builder
	if m.err != nil {
		b.WriteString("⚠ " + m.err.Error())
	} else {
		b.WriteString(m.s.line())
	}
	b.WriteString("\n")
	if m.help {
		for _, l := range helpLines(m.keys) {
			fmt.Fprintf(&b, "  %-22s %s\n", l[0], l[1])
		}
		b.WriteString("\n  press any key")
		return b.String()
	}
	if len(m.s.List) == 0 {
		b.WriteString("\n  playlist is empty — add music with: wimamp add ~/Music   (? for keys)\n")
	}
	// rows to show: the whole playlist, or the matches while searching
	idx, sel, top, h := make([]int, len(m.s.List)), m.cursor, m.top, m.h
	for i := range idx {
		idx[i] = i
	}
	if m.srch.on {
		h--
		idx, sel = match(m.s.List, m.srch.q), m.srch.sel
		top = max(0, sel-h+1)
	}
	for n := top; n < len(idx) && n < top+h; n++ {
		e := m.s.List[idx[n]]
		mark := "  "
		if e.Current {
			mark = "▶ "
		}
		line := fmt.Sprintf("%s%3d. %s", mark, idx[n]+1, e.name())
		if n == sel {
			line = "\x1b[7m" + line + "\x1b[0m"
		}
		b.WriteString(line + "\n")
	}
	if m.srch.on {
		fmt.Fprintf(&b, "/%s█  %d matches", m.srch.q, len(idx))
	}
	if m.flash != "" {
		b.WriteString("\n" + m.flash)
	}
	return b.String()
}
