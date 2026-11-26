package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// The TUI's file browser: one folder's subfolders and audio files in the playlist panel.

type browseItem struct {
	name string
	dir  bool
}

type browser struct {
	on     bool
	dir    string
	all    []browseItem // the folder's entries
	items  []browseItem // those matching q, as listed
	sel    int
	marks  []string // full paths, in the order marked
	q      string   // search text: names containing it, ignoring case
	typing bool     // keys go to the search text
	lists  bool     // choosing a saved playlist to load, not music to add
}

var listExt = map[string]bool{".m3u": true, ".m3u8": true}

// filter lists the entries matching q and selects the first.
func (b *browser) filter() {
	b.items, b.sel = b.all, 0
	if b.q != "" {
		q := strings.ToLower(b.q)
		b.items = nil
		for _, it := range b.all {
			if strings.Contains(strings.ToLower(it.name), q) {
				b.items = append(b.items, it)
			}
		}
	}
}

// searchKey edits the search text; done is true when the key should then act as usual (enter).
func (b *browser) searchKey(k string) (done bool) {
	switch k {
	case "esc":
		b.q, b.typing = "", false
	case "enter":
		b.typing = false
		return true
	case "backspace":
		if rs := []rune(b.q); len(rs) > 0 {
			b.q = string(rs[:len(rs)-1])
		}
	case "down", "ctrl+n":
		b.sel = min(b.sel+1, max(len(b.items)-1, 0))
		return false
	case "up", "ctrl+p":
		b.sel = max(b.sel-1, 0)
		return false
	case "tab":
		return true // marks, as outside the search
	case "space":
		b.q += " "
	default:
		if utf8.RuneCountInString(k) != 1 {
			return false
		}
		b.q += k
	}
	b.filter()
	return false
}

// listDir is a folder's subfolders, then its files with an extension in exts, each sorted ignoring case; hidden ones skipped.
// shortcut: "hidden" means a leading dot, so Windows hidden-attribute files show; check the attribute if that annoys.
func listDir(dir string, exts map[string]bool) ([]browseItem, error) {
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
		if isDir || exts[strings.ToLower(filepath.Ext(n))] {
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
	exts := audioExt
	if b.lists {
		exts = listExt
	}
	items, err := listDir(dir, exts)
	if err != nil {
		return err
	}
	b.dir, b.all, b.q, b.typing = dir, items, "", false
	b.filter()
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
		m.br.lists = false
		if err = m.br.cd(d, ""); err == nil {
			m.br.on, m.br.marks = true, nil
			m.srch.on, m.queueOn = false, false
			return
		}
	}
	m.flash = err.Error()
}

// openLists shows the browser on the saved-playlists folder, to load one.
func (m *model) openLists() {
	dir := filepath.Join(configDir(), "playlists")
	m.br.lists = true
	err := os.MkdirAll(dir, 0o755)
	if err == nil {
		err = m.br.cd(dir, "")
	}
	if err != nil {
		m.flash = err.Error()
		return
	}
	m.br.on, m.br.marks = true, nil
	m.srch.on, m.queueOn = false, false
}

// closeBrowser hides it and keeps its folder in the player for next time, then adds paths if any.
func (m *model) closeBrowser(add []string) tea.Cmd {
	m.br.on = false
	if m.br.lists { // the playlists folder is not where music is browsed from
		if len(add) == 0 {
			return nil
		}
		return run(func() error { return loadList(add[0]) })
	}
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
	if b.typing && !b.searchKey(k) {
		return nil
	}
	var err error
	onDir := b.sel < len(b.items) && b.items[b.sel].dir
	switch k {
	case "esc":
		if b.q != "" { // first esc clears the search
			b.q = ""
			b.filter()
			return nil
		}
		return m.closeBrowser(nil)
	case "/":
		b.typing = true
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
		if b.sel < len(b.items) && !b.lists { // one playlist is loaded, not several
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
	prefix := ""
	if b.lists {
		prefix = "LOAD "
	}
	if over := cols(prefix+title) - (w - 6); over > 0 { // keep the end of the path, the part that changes
		title = "…" + cutLeft(title, over+1)
	}
	title = prefix + title
	hint := stDim.Render("/ search  space mark  a add  h up  esc back")
	if b.lists {
		hint = stDim.Render("/ search  enter load  h up  esc back")
	}
	if len(b.marks) > 0 {
		hint = stGold.Render(fmt.Sprintf("%d marked", len(b.marks))) + stDim.Render("  a add  esc back")
	}
	switch {
	case b.typing:
		hint = stPlay.Render(fmt.Sprintf("/%s█  %d matches", b.q, len(b.items))) + stDim.Render("  tab mark  enter open")
	case b.q != "":
		hint = stPlay.Render(fmt.Sprintf("/%s  %d matches", b.q, len(b.items))) + stDim.Render("  esc clear")
	}
	if m.flash != "" {
		hint = stErr.Render(m.flash)
	}
	lines := []string{boxTop(title, "", w)}
	switch {
	case len(b.items) == 0 && b.q != "":
		lines = append(lines, boxRow(stDim.Render("nothing matches"), w))
	case len(b.items) == 0:
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
