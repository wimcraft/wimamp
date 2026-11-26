package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// id3 is a minimal ID3v2.3 tag with text frames, followed by a little filler audio.
func id3(frames map[string]string) []byte {
	var body bytes.Buffer
	for _, id := range []string{"TIT2", "TPE1", "TALB", "TYER", "TRCK", "TCON"} {
		v, ok := frames[id]
		if !ok {
			continue
		}
		data := append([]byte{0}, v...) // 0: ISO-8859-1
		body.WriteString(id)
		binary.Write(&body, binary.BigEndian, uint32(len(data)))
		body.Write([]byte{0, 0})
		body.Write(data)
	}
	n := body.Len()
	head := []byte{'I', 'D', '3', 3, 0, 0, byte(n >> 21 & 0x7f), byte(n >> 14 & 0x7f), byte(n >> 7 & 0x7f), byte(n & 0x7f)}
	return append(append(head, body.Bytes()...), make([]byte, 64)...)
}

// flacWithComments is a FLAC header: STREAMINFO, then a Vorbis comment block.
func flacWithComments(comments ...string) []byte {
	var b bytes.Buffer
	b.WriteString("fLaC")
	b.Write([]byte{0, 0, 0, 34})
	b.Write(make([]byte, 34))
	var vc bytes.Buffer
	le := func(n int) { binary.Write(&vc, binary.LittleEndian, uint32(n)) }
	le(len("wimamp"))
	vc.WriteString("wimamp")
	le(len(comments))
	for _, c := range comments {
		le(len(c))
		vc.WriteString(c)
	}
	n := vc.Len()
	b.Write([]byte{0x80 | 4, byte(n >> 16), byte(n >> 8), byte(n)}) // last block, type 4
	b.Write(vc.Bytes())
	return b.Bytes()
}

func TestInfoRows(t *testing.T) {
	dir := t.TempDir()
	mp3 := filepath.Join(dir, "a.mp3")
	os.WriteFile(mp3, id3(map[string]string{"TIT2": "Stop!", "TPE1": "Jane's Addiction", "TALB": "Ritual", "TYER": "1990", "TRCK": "3/9"}), 0o644)
	flac := filepath.Join(dir, "b.flac")
	os.WriteFile(flac, flacWithComments("TITLE=Intro", "ARTIST=Someone", "GENRE=Jazz"), 0o644)
	wav := filepath.Join(dir, "c.wav")
	silentWAV(t, wav, 0.1)

	get := func(rows [][2]string, l string) string {
		for _, r := range rows {
			if r[0] == l {
				return r[1]
			}
		}
		return ""
	}
	r := infoRows(entry{Filename: mp3}, "3:06")
	for l, want := range map[string]string{"Title": "Stop!", "Artist": "Jane's Addiction", "Album": "Ritual", "Year": "1990", "Track": "3/9", "Length": "3:06", "Format": "MP3", "File": "a.mp3", "Folder": dir} {
		if got := get(r, l); got != want {
			t.Errorf("mp3 %s = %q, want %q (rows %v)", l, got, want, r)
		}
	}
	if get(r, "Genre") != "" || get(r, "Error") != "" {
		t.Errorf("mp3 rows %v", r)
	}
	r = infoRows(entry{Filename: flac}, "")
	if get(r, "Title") != "Intro" || get(r, "Genre") != "Jazz" || get(r, "Format") != "FLAC" || get(r, "Length") != "" {
		t.Errorf("flac rows %v", r)
	}
	r = infoRows(entry{Filename: wav}, "0:00")
	var labels []string
	for _, x := range r {
		labels = append(labels, x[0])
	}
	if !slices.Equal(labels, []string{"Length", "Format", "File", "Folder"}) {
		t.Errorf("untagged rows %v", r)
	}
	r = infoRows(entry{Filename: filepath.Join(dir, "gone.mp3")}, "")
	if !strings.Contains(get(r, "Error"), "no such file") || get(r, "File") != "gone.mp3" {
		t.Errorf("missing file rows %v", r)
	}
	if r = infoRows(entry{Filename: "https://radio/x"}, ""); len(r) != 1 || r[0][0] != "URL" {
		t.Errorf("url rows %v", r)
	}
}

func TestRevealCmd(t *testing.T) {
	c := revealCmd("/m/a b/x.mp3")
	want := map[string][]string{"darwin": {"open", "-R", "/m/a b/x.mp3"}, "windows": {"explorer", "/select,/m/a b/x.mp3"}}[runtime.GOOS]
	if want == nil {
		want = []string{"xdg-open", "/m/a b"}
	}
	if !slices.Equal(c.Args, want) {
		t.Errorf("args %v, want %v", c.Args, want)
	}
	if reveal(entry{Filename: "http://x"}) == nil {
		t.Error("revealing a URL")
	}
}

func TestInfoView(t *testing.T) {
	m := testModel(40, 14)
	m.keys, _ = parseKeys("")
	m.cursor = 1
	cmd := m.do("info")
	if m.info == nil || m.info.e.ID != 2 || cmd == nil {
		t.Fatalf("i did not open the info view: %+v", m.info)
	}
	if !strings.Contains(strings.Join(screen(m), "\n"), "reading…") {
		t.Error("no reading line before the tags arrive")
	}
	m.Update(infoMsg{2, [][2]string{{"Title", "Stop!"}, {"Folder", "/a/very/long/folder/name/that/must/wrap/around"}}})
	lines := screen(m)
	checkSize(t, lines, 40, 14)
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "TRACK INFO") || !strings.Contains(text, "Stop!") || !strings.Contains(text, "wrap/around") {
		t.Errorf("info view:\n%s", text)
	}
	m.key("j") // list navigation does nothing here
	m.key("i")
	if m.info != nil || m.cursor != 1 {
		t.Errorf("i again: info %v, cursor %d", m.info, m.cursor)
	}
}

func TestListKeysFromInfo(t *testing.T) {
	m := testModel(40, 14)
	m.keys, _ = parseKeys("")
	m.info = &infoView{e: m.s.List[0]}
	m.key("ctrl+s")
	if m.info != nil || !m.saving {
		t.Errorf("ctrl+s from info: info %v saving %v", m.info, m.saving)
	}
}
