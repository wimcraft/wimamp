package main

import (
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg struct{}
type stateMsg struct {
	s   state
	err error
}
type flashMsg string
type lengthsMsg struct{} // background length reads finished a batch: just re-render

type model struct {
	s              state
	err            error
	flash, pending string
	keys           map[string]string
	cursor, top, h int
	w              int
	synced, help   bool
	srch           search
	queueOn        bool // the list shows the queue
	qsel           int
	br             browser   // the file browser, shown in place of the list
	info           *infoView // the track info view, shown in place of the list
	saving         bool      // the bottom border asks for a name to save the playlist as
	saveName       string
}

type infoMsg struct {
	id   int
	rows [][2]string
}

func runTUI() error {
	keys, err := loadKeys()
	if err != nil {
		return err
	}
	if err := ensureDaemon(); err != nil {
		return err
	}
	p := tea.NewProgram(&model{keys: keys, h: 20}, tea.WithAltScreen())
	lengths.wake = func() { p.Send(lengthsMsg{}) }
	_, err = p.Run()
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
		m.h, m.w = max(msg.Height-playerLines-2, 1), msg.Width // the player box and the playlist borders
	case tickMsg:
		return m, tea.Batch(refresh, tick())
	case flashMsg:
		m.flash = string(msg)
	case infoMsg:
		if m.info != nil && m.info.e.ID == msg.id {
			m.info.rows = msg.rows
		}
	case stateMsg:
		if !reflect.DeepEqual(msg.s.List, m.s.List) {
			go queueLengths(paths(msg.s.List), false)
		}
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
	case m.saving:
		return m.saveKey(k)
	case m.info != nil:
		return m.infoKey(k)
	case m.br.on:
		return m.browseKey(k)
	case m.srch.on:
		if i, queue := m.srch.key(k, m.s.List); queue {
			return run(playerAction("queue", i, len(m.s.List)))
		} else if i >= 0 {
			m.cursor = i
			return run(playerAction("play", i, len(m.s.List)))
		}
	case k == "d" && m.queueOn: // a single d unqueues here: nothing in the queue view is deleted for good
		return m.do("remove")
	case k == "esc" && m.queueOn:
		m.queueOn = false
	default:
		return m.do(resolve(m.keys, &m.pending, k))
	}
	return nil
}

func (m *model) do(a string) tea.Cmd {
	cursor := m.cursor
	if m.queueOn {
		idx := queueIndices(m.s.List, m.s.Queue)
		if c, ok := moveCursor(a, m.qsel, m.h, rowEntries(m.s.List, idx)); ok {
			m.qsel = c
			return nil
		}
		a = queueAction(a)
		if m.qsel >= len(idx) && onRow[a] {
			return nil
		}
		if m.qsel < len(idx) {
			cursor = idx[m.qsel]
		}
		m.qsel = moved(a, m.qsel, len(idx)) // the selection follows the moved entry
	} else if c, ok := moveCursor(a, m.cursor, m.h, m.s.List); ok {
		m.cursor = c
		return nil
	} else {
		m.cursor = moved(a, m.cursor, len(m.s.List))
	}
	switch a {
	case "queue-view":
		m.queueOn, m.qsel = !m.queueOn, 0
	case "search":
		m.srch, m.queueOn = search{on: true}, false
	case "help":
		m.help = true
	case "add-files", "add-folder":
		m.openBrowser()
	case "new-list":
		return run(clearList)
	case "save-list":
		m.saving, m.saveName = true, ""
	case "load-list":
		m.openLists()
	case "info":
		if cursor < len(m.s.List) {
			e, length := m.s.List[cursor], rowLength(m.s.List[cursor], m.s)
			m.info = &infoView{e: e}
			return func() tea.Msg { return infoMsg{e.ID, infoRows(e, length)} } // reads the file: off the UI loop
		}
	case "quit":
		return tea.Quit
	case "quit-player":
		return tea.Sequence(func() tea.Msg { call(simple["quit"]); return nil }, tea.Quit)
	}
	if f := playerAction(a, cursor, len(m.s.List)); f != nil {
		return run(f)
	}
	return nil // e.g. pin: window only
}

func (m *model) clamp() {
	m.qsel = max(min(m.qsel, len(queueIndices(m.s.List, m.s.Queue))-1), 0)
	m.cursor = max(min(m.cursor, len(m.s.List)-1), 0)
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+m.h {
		m.top = m.cursor - m.h + 1
	}
}

func (m *model) rowName(e entry) string {
	if m.srch.on {
		return searchName(e, m.srch.q)
	}
	return e.name()
}

// infoKey handles a key while the info view is shown: player controls work, the list doesn't.
func (m *model) infoKey(k string) tea.Cmd {
	if k == "esc" {
		m.info = nil
		return nil
	}
	switch a := resolve(m.keys, &m.pending, k); {
	case a == "info":
		m.info = nil
	case a == "reveal":
		e := m.info.e
		return func() tea.Msg {
			if err := reveal(e); err != nil {
				return flashMsg(err.Error())
			}
			return nil
		}
	case listActions[a]: // they change the list the info came from: leave the view first
		m.info = nil
		return m.do(a)
	case infoActions[a]:
		return m.do(a)
	}
	return nil
}

// saveKey edits the name the playlist is saved as: a bare name goes to the saved-playlists folder.
func (m *model) saveKey(k string) tea.Cmd {
	switch k {
	case "esc":
		m.saving = false
	case "enter":
		name := strings.TrimSpace(m.saveName)
		if name == "" {
			return nil
		}
		m.saving = false
		p := playlistPath(name)
		return run(func() error { return save(p) })
	case "backspace":
		if rs := []rune(m.saveName); len(rs) > 0 {
			m.saveName = string(rs[:len(rs)-1])
		}
	case "space":
		m.saveName += " "
	default:
		if utf8.RuneCountInString(k) == 1 {
			m.saveName += k
		}
	}
	return nil
}
