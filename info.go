package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dhowden/tag"
)

// Track info: tags read from the file itself, so any playlist entry works, playing or not.

// infoView is the entry the info view shows and its label/value rows (nil until read).
type infoView struct {
	e    entry
	rows [][2]string
}

// infoRows reads e's tags and returns the rows to show; length is the entry's known length or "".
// Missing tags are left out; a file that can't be read shows the error.
func infoRows(e entry, length string) [][2]string {
	p := e.Filename
	if strings.Contains(p, "://") {
		return [][2]string{{"URL", p}}
	}
	var rows [][2]string
	add := func(label, v string) {
		if v = strings.TrimSpace(v); v != "" && v != "0" {
			rows = append(rows, [2]string{label, v})
		}
	}
	format := strings.ToUpper(strings.TrimPrefix(filepath.Ext(p), "."))
	f, err := os.Open(p)
	if err == nil {
		m, terr := tag.ReadFrom(f)
		f.Close()
		switch {
		case terr == nil:
			n, of := m.Track()
			d, dof := m.Disc()
			add("Title", m.Title())
			add("Artist", m.Artist())
			add("Album", m.Album())
			add("Album artist", m.AlbumArtist())
			add("Year", fmt.Sprint(m.Year()))
			add("Track", count(n, of))
			add("Disc", count(d, dof))
			add("Genre", m.Genre())
			if t := string(m.FileType()); t != "" && t != string(tag.UnknownFileType) {
				format = t
			}
		case !errors.Is(terr, tag.ErrNoTagsFound):
			err = terr
		}
	}
	add("Length", length)
	add("Format", format)
	add("File", filepath.Base(p))
	add("Folder", filepath.Dir(p))
	if err != nil {
		add("Error", err.Error())
	}
	return rows
}

// count is "3/12", "3", or "" for a track or disc number.
func count(n, of int) string {
	switch {
	case n <= 0:
		return ""
	case of > 0:
		return fmt.Sprintf("%d/%d", n, of)
	}
	return fmt.Sprint(n)
}

// revealCmd opens the system file manager at p: the file selected on macOS and Windows, its folder elsewhere.
func revealCmd(p string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", p)
	case "windows":
		return exec.Command("explorer", "/select,"+p)
	}
	return exec.Command("xdg-open", filepath.Dir(p))
}

// reveal starts the file manager; only a failure to start is reported (explorer exits 1 even on success).
func reveal(e entry) error {
	if strings.Contains(e.Filename, "://") {
		return errors.New("not a file: " + e.Filename)
	}
	c := revealCmd(e.Filename)
	if err := c.Start(); err != nil {
		return fmt.Errorf("show in folder: %w", err)
	}
	go c.Wait()
	return nil
}

// infoActions are the keys that still work while the info view is shown: player controls, not the list.
var infoActions = map[string]bool{
	"resume": true, "toggle": true, "stop": true, "next": true, "prev": true, "seek-forward": true, "seek-back": true,
	"volume-up": true, "volume-down": true, "shuffle": true, "repeat": true, "help": true, "quit": true, "quit-player": true, "pin": true,
}
