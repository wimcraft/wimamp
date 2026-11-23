package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// The TUI's file browser: one folder's subfolders and audio files in the playlist panel.

type browseItem struct {
	name string
	dir  bool
}

type browser struct {
	on    bool
	dir   string
	items []browseItem
	sel   int
	marks []string // full paths, in the order marked
}

// listDir is a folder's subfolders, then its audio files, each sorted ignoring case; hidden ones skipped.
// shortcut: "hidden" means a leading dot, so Windows hidden-attribute files show; check the attribute if that annoys.
func listDir(dir string) ([]browseItem, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []browseItem
	for _, e := range ents {
		n := e.Name()
		if strings.HasPrefix(n, ".") {
			continue
		}
		isDir := e.IsDir()
		if e.Type()&fs.ModeSymlink != 0 { // follow links; broken ones are skipped
			st, err := os.Stat(filepath.Join(dir, n))
			if err != nil {
				continue
			}
			isDir = st.IsDir()
		}
		if isDir || audioExt[strings.ToLower(filepath.Ext(n))] {
			out = append(out, browseItem{n, isDir})
		}
	}
	slices.SortFunc(out, func(a, b browseItem) int {
		if a.dir != b.dir {
			if a.dir {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
	})
	return out, nil
}

// cd shows dir with the entry named sel selected (or the first); on error nothing changes.
func (b *browser) cd(dir, sel string) error {
	items, err := listDir(dir)
	if err != nil {
		return err
	}
	b.dir, b.items, b.sel = dir, items, 0
	for i, it := range items {
		if it.name == sel {
			b.sel = i
		}
	}
	return nil
}

func (b *browser) path(i int) string { return filepath.Join(b.dir, b.items[i].name) }

func (b *browser) marked(p string) bool { return slices.Contains(b.marks, p) }

// take is what adding appends: the marks in order, or the selected entry when nothing is marked.
func (b *browser) take() []string {
	if len(b.marks) > 0 {
		return b.marks
	}
	if b.sel < len(b.items) {
		return []string{b.path(b.sel)}
	}
	return nil
}

// openBrowser starts in the folder last left, else the playing track's folder, else home.
func (m *model) openBrowser() {
	home, _ := os.UserHomeDir()
	starts := []string{m.s.BrowseDir}
	if i := currentIndex(m.s.List, -1); i >= 0 && !strings.Contains(m.s.List[i].Filename, "://") {
		starts = append(starts, filepath.Dir(m.s.List[i].Filename))
	}
	var err error
	for _, d := range append(starts, home) {
		if d == "" {
			continue
		}
		if err = m.br.cd(d, ""); err == nil {
			m.br.on, m.br.marks = true, nil
			m.srch.on, m.queueOn = false, false
			return
		}
	}
	m.flash = err.Error()
}

// closeBrowser hides it and keeps its folder in the player for next time, then adds paths if any.
func (m *model) closeBrowser(add []string) tea.Cmd {
	m.br.on = false
	dir := m.br.dir
	m.s.BrowseDir = dir
	return run(func() error {
		if _, err := call([]any{"set_property", "user-data/wimamp/browse-dir", dir}); err != nil {
			return err
		}
		if len(add) == 0 {
			return nil
		}
		return load(add, "append")
	})
}

// browseKey handles a key while the browser is open; other player keys do nothing here.
func (m *model) browseKey(k string) tea.Cmd {
	b := &m.br
	var err error
	onDir := b.sel < len(b.items) && b.items[b.sel].dir
	switch k {
	case "esc":
		return m.closeBrowser(nil)
	case "?":
		m.help = true
	case "q":
		return tea.Quit
	case "l", "right":
		if onDir {
			err = b.cd(b.path(b.sel), "")
		}
	case "enter":
		if onDir {
			err = b.cd(b.path(b.sel), "")
		} else if b.sel < len(b.items) {
			return m.closeBrowser(b.take())
		}
	case "a", "A":
		return m.closeBrowser(b.take())
	case "h", "left", "backspace":
		if up := filepath.Dir(b.dir); up != b.dir {
			err = b.cd(up, filepath.Base(b.dir))
		}
	case "~":
		home, _ := os.UserHomeDir()
		err = b.cd(home, "")
	case "space", "tab":
		if b.sel < len(b.items) {
			p := b.path(b.sel)
			if i := slices.Index(b.marks, p); i >= 0 {
				b.marks = slices.Delete(b.marks, i, i+1)
			} else {
				b.marks = append(b.marks, p)
			}
			b.sel = min(b.sel+1, len(b.items)-1)
		}
	default: // list navigation through the user's bindings
		if c, ok := moveCursor(resolve(m.keys, &m.pending, k), b.sel, m.h, make([]entry, len(b.items))); ok {
			b.sel = c
		}
	}
	if err != nil {
		m.flash = err.Error()
	}
	return nil
}

// browseBox draws the browser in the playlist box: m.h rows between its borders.
func (m *model) browseBox(w int) []string {
	b, in, h := &m.br, w-4, m.h
	title := b.dir
	if home, _ := os.UserHomeDir(); home != "" && strings.HasPrefix(title+"/", home+"/") {
		title = "~" + strings.TrimPrefix(title, home)
	}
	if over := cols(title) - (w - 6); over > 0 { // keep the end of the path, the part that changes
		title = "…" + cutLeft(title, over+1)
	}
	hint := stDim.Render("space mark  a add  h up  esc back")
	if len(b.marks) > 0 {
		hint = stGold.Render(fmt.Sprintf("%d marked", len(b.marks))) + stDim.Render("  a add  esc back")
	}
	if m.flash != "" {
		hint = stErr.Render(m.flash)
	}
	lines := []string{boxTop(title, "", w)}
	if len(b.items) == 0 {
		lines = append(lines, boxRow(stDim.Render("no folders or music here — h goes up"), w))
	}
	top := max(0, b.sel-h+1)
	for n := top; n < len(b.items) && n < top+h; n++ {
		it, mark := b.items[n], "  "
		if b.marked(b.path(n)) {
			mark = "✓ "
		}
		name := it.name
		if it.dir {
			name += "/"
		}
		row := padTo(mark+name, in)
		st := stText
		if it.dir {
			st = stPlay
		}
		if n == b.sel {
			lines = append(lines, border("│")+st.Background(selBg).Render(" "+row+" ")+border("│"))
			continue
		}
		lines = append(lines, boxRow(st.Render(row), w))
	}
	for len(lines) < h+1 {
		lines = append(lines, boxRow("", w))
	}
	return append(lines[:h+1], boxBottom(hint, w))
}
