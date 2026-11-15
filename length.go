package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Track lengths, read from file headers (never a full scan) and kept in memory only.

// trackLength reads a file's length in seconds from its header; ok is false for
// other formats, URLs, and missing or damaged files.
func trackLength(path string) (float64, bool) {
	read := lengthReaders[strings.ToLower(filepath.Ext(path))]
	if read == nil || strings.Contains(path, "://") {
		return 0, false
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return 0, false
	}
	sec, ok := read(f, st.Size())
	if !ok || sec <= 0 || math.IsInf(sec, 0) || math.IsNaN(sec) || sec > 1e7 {
		return 0, false
	}
	return sec, true
}

var lengthReaders = map[string]func(io.ReaderAt, int64) (float64, bool){
	".mp3": mp3Length, ".flac": flacLength, ".wav": wavLength, ".aif": aiffLength, ".aiff": aiffLength,
	".ogg": oggLength, ".oga": oggLength, ".opus": oggLength, ".m4a": mp4Length, ".mp4": mp4Length, ".alac": mp4Length,
}

// at reads n bytes at off, or nil when the file is shorter.
func at(r io.ReaderAt, off int64, n int) []byte {
	if off < 0 || n < 0 {
		return nil
	}
	b := make([]byte, n)
	if _, err := r.ReadAt(b, off); err != nil {
		return nil
	}
	return b
}

// id3End is where audio starts after an ID3v2 tag (0 when there is none).
func id3End(r io.ReaderAt) int64 {
	h := at(r, 0, 10)
	if h == nil || string(h[:3]) != "ID3" {
		return 0
	}
	n := int64(h[6]&0x7f)<<21 | int64(h[7]&0x7f)<<14 | int64(h[8]&0x7f)<<7 | int64(h[9]&0x7f)
	if h[5]&0x10 != 0 { // footer
		n += 10
	}
	return 10 + n
}

func flacLength(r io.ReaderAt, _ int64) (float64, bool) {
	o := id3End(r)
	b := at(r, o, 42) // "fLaC", block header, STREAMINFO (always first)
	if b == nil || string(b[:4]) != "fLaC" || b[4]&0x7f != 0 {
		return 0, false
	}
	d := b[8:]
	rate := int(d[10])<<12 | int(d[11])<<4 | int(d[12])>>4
	total := uint64(d[13]&0x0f)<<32 | uint64(binary.BigEndian.Uint32(d[14:18]))
	if rate == 0 {
		return 0, false
	}
	return float64(total) / float64(rate), true
}

// chunks walks RIFF/IFF chunks from off, calling f with each id, data offset and size until it returns true.
func chunks(r io.ReaderAt, off, size int64, order binary.ByteOrder, f func(id string, data, n int64) bool) {
	for i := 0; i < 1000 && off+8 <= size; i++ {
		h := at(r, off, 8)
		if h == nil {
			return
		}
		n := int64(order.Uint32(h[4:]))
		if f(string(h[:4]), off+8, min(n, size-off-8)) {
			return
		}
		off += 8 + n + n&1
	}
}

func wavLength(r io.ReaderAt, size int64) (float64, bool) {
	h := at(r, 0, 12)
	if h == nil || string(h[:4]) != "RIFF" || string(h[8:]) != "WAVE" {
		return 0, false
	}
	var rate, data int64
	chunks(r, 12, size, binary.LittleEndian, func(id string, off, n int64) bool {
		switch id {
		case "fmt ":
			if b := at(r, off+8, 4); b != nil {
				rate = int64(binary.LittleEndian.Uint32(b))
			}
		case "data":
			data = n
			return true
		}
		return false
	})
	if rate == 0 || data == 0 {
		return 0, false
	}
	return float64(data) / float64(rate), true
}

func aiffLength(r io.ReaderAt, size int64) (float64, bool) {
	h := at(r, 0, 12)
	if h == nil || string(h[:4]) != "FORM" || (string(h[8:]) != "AIFF" && string(h[8:]) != "AIFC") {
		return 0, false
	}
	var sec float64
	chunks(r, 12, size, binary.BigEndian, func(id string, off, n int64) bool {
		if id != "COMM" {
			return false
		}
		if b := at(r, off, 18); b != nil {
			frames := float64(binary.BigEndian.Uint32(b[2:6]))
			// 80-bit extended float: 15-bit exponent, 64-bit mantissa with explicit integer bit
			exp := int(binary.BigEndian.Uint16(b[8:10])&0x7fff) - 16383 - 63
			rate := math.Ldexp(float64(binary.BigEndian.Uint64(b[10:18])), exp)
			if rate > 0 {
				sec = frames / rate
			}
		}
		return true
	})
	return sec, sec > 0
}

func oggLength(r io.ReaderAt, size int64) (float64, bool) {
	h := at(r, 0, 27)
	if h == nil || string(h[:4]) != "OggS" {
		return 0, false
	}
	p := at(r, 27+int64(h[26]), 19) // first packet: the codec's identification header
	if p == nil {
		return 0, false
	}
	var rate, skip float64
	switch {
	case string(p[:7]) == "\x01vorbis":
		rate = float64(binary.LittleEndian.Uint32(p[12:16]))
	case string(p[:8]) == "OpusHead":
		rate, skip = 48000, float64(binary.LittleEndian.Uint16(p[10:12]))
	default:
		return 0, false
	}
	n := min(size, 64<<10)
	tail := at(r, size-n, int(n))
	for i := bytes.LastIndex(tail, []byte("OggS")); i >= 0; i = bytes.LastIndex(tail[:i], []byte("OggS")) {
		if i+14 > len(tail) {
			continue
		}
		g := int64(binary.LittleEndian.Uint64(tail[i+6:]))
		if g > 0 && rate > 0 {
			return (float64(g) - skip) / rate, true
		}
	}
	return 0, false
}

// mp4Length reads moov/mvhd, walking top-level boxes by size so a moov at the end costs no extra reads.
func mp4Length(r io.ReaderAt, size int64) (float64, bool) {
	box := func(off, end int64, want string) (int64, int64) { // data offset and end of the first box named want
		for i := 0; i < 1000 && off+8 <= end; i++ {
			h := at(r, off, 16)
			if h == nil {
				h = at(r, off, 8)
			}
			if h == nil {
				return -1, 0
			}
			n, hl := int64(binary.BigEndian.Uint32(h)), int64(8)
			switch {
			case n == 1 && len(h) == 16:
				n, hl = int64(binary.BigEndian.Uint64(h[8:])), 16
			case n == 0:
				n = end - off
			}
			if n < hl {
				return -1, 0
			}
			if string(h[4:8]) == want {
				return off + hl, min(off+n, end)
			}
			off += n
		}
		return -1, 0
	}
	moov, moovEnd := box(0, size, "moov")
	if moov < 0 {
		return 0, false
	}
	mvhd, _ := box(moov, moovEnd, "mvhd")
	b := at(r, mvhd, 32)
	if mvhd < 0 || b == nil {
		return 0, false
	}
	var scale, dur float64
	if b[0] == 1 {
		scale, dur = float64(binary.BigEndian.Uint32(b[20:24])), float64(binary.BigEndian.Uint64(b[24:32]))
	} else {
		scale, dur = float64(binary.BigEndian.Uint32(b[12:16])), float64(binary.BigEndian.Uint32(b[16:20]))
	}
	if scale == 0 {
		return 0, false
	}
	return dur / scale, true
}

var mp3Bitrates = [5][16]int{ // kbps: MPEG1 L1, L2, L3; MPEG2/2.5 L1; MPEG2/2.5 L2 and L3
	{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448},
	{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},
	{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},
	{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
	{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
}

func mp3Length(r io.ReaderAt, size int64) (float64, bool) {
	start := id3End(r)
	buf := at(r, start, int(min(size-start, 64<<10)))
	for i := 0; i+4 <= len(buf); i++ {
		b := buf[i:]
		ver, layer := b[1]>>3&3, b[1]>>1&3 // ver 3 = MPEG1, 2 = MPEG2, 0 = MPEG2.5; layer 3 = L1 … 1 = L3
		bri, sri := b[2]>>4, b[2]>>2&3
		if b[0] != 0xff || b[1]&0xe0 != 0xe0 || ver == 1 || layer == 0 || bri == 0 || bri == 15 || sri == 3 {
			continue
		}
		rate := [3]int{44100, 48000, 32000}[sri]
		row, spf := 4, 1152
		switch {
		case ver == 3:
			row = int(3 - layer)
		case layer == 3:
			row = 3
		}
		if ver != 3 {
			rate >>= 1
			if ver == 0 {
				rate >>= 1
			}
			if layer == 1 {
				spf = 576
			}
		}
		if layer == 3 {
			spf = 384
		}
		side := 32 // side info, where a Xing/Info header follows
		mono := b[3]>>6 == 3
		switch {
		case ver == 3 && mono, ver != 3 && !mono:
			side = 17
		case ver != 3:
			side = 9
		}
		if size-start-int64(i) < int64(spf/8*mp3Bitrates[row][bri]*1000/rate) { // not even one whole frame
			return 0, false
		}
		frames := 0
		if x := at(r, start+int64(i)+4+int64(side), 12); x != nil && (string(x[:4]) == "Xing" || string(x[:4]) == "Info") && x[7]&1 != 0 {
			frames = int(binary.BigEndian.Uint32(x[8:]))
		} else if v := at(r, start+int64(i)+36, 18); v != nil && string(v[:4]) == "VBRI" {
			frames = int(binary.BigEndian.Uint32(v[14:]))
		}
		if frames > 0 {
			return float64(frames) * float64(spf) / float64(rate), true
		}
		audio := size - start - int64(i)
		if t := at(r, size-128, 3); t != nil && string(t) == "TAG" {
			audio -= 128
		}
		// shortcut: CBR estimate from the first frame; off for VBR files without a Xing/VBRI header (mpv corrects it on play)
		return float64(audio) * 8 / float64(mp3Bitrates[row][bri]*1000), true
	}
	return 0, false
}

// The cache: one entry per path, valid while the file's size and mtime are unchanged.

type lengthEntry struct {
	size      int64
	mtime     time.Time
	sec       float64
	ok, exact bool // exact: reported by mpv during playback, never overwritten by a header read
}

var lengths = struct {
	sync.Mutex
	m      map[string]lengthEntry
	q      []string
	queued map[string]bool
	kick   chan struct{}
	wake   func() // set once by the interface; called when new lengths are known
	once   sync.Once
}{m: map[string]lengthEntry{}, queued: map[string]bool{}, kick: make(chan struct{}, 1)}

// cachedLength is a map lookup only, safe to call while drawing.
func cachedLength(path string) (sec float64, ok, known bool) {
	lengths.Lock()
	defer lengths.Unlock()
	e, known := lengths.m[path]
	return e.sec, e.ok, known
}

// learnLength records the length mpv reports for the playing file.
func learnLength(path string, sec float64) {
	st, err := os.Stat(path)
	if err != nil || sec <= 0 {
		return
	}
	lengths.Lock()
	defer lengths.Unlock()
	if e := lengths.m[path]; e.exact && e.sec == sec && e.size == st.Size() && e.mtime.Equal(st.ModTime()) {
		return
	}
	lengths.m[path] = lengthEntry{st.Size(), st.ModTime(), sec, true, true}
}

// queueLengths asks the background worker to read paths: front ones (rows on screen) before everything queued.
func queueLengths(paths []string, front bool) {
	lengths.Lock()
	var add []string
	for _, p := range paths {
		if !lengths.queued[p] {
			lengths.queued[p] = true
			add = append(add, p)
		} else if front { // already queued at the back: move it up
			for i, q := range lengths.q {
				if q == p {
					lengths.q = append(lengths.q[:i], lengths.q[i+1:]...)
					break
				}
			}
			add = append(add, p)
		}
	}
	if front {
		lengths.q = append(add, lengths.q...)
	} else {
		lengths.q = append(lengths.q, add...)
	}
	lengths.Unlock()
	if len(add) > 0 {
		lengths.once.Do(func() { go lengthWorker() })
		select {
		case lengths.kick <- struct{}{}:
		default:
		}
	}
}

func paths(list []entry) []string {
	ps := make([]string, len(list))
	for i, e := range list {
		ps[i] = e.Filename
	}
	return ps
}

// queueMissing queues the paths of list that the cache has no entry for (visible rows: front).
func queueMissing(list []entry, front bool) {
	var ps []string
	for _, e := range list {
		if _, _, known := cachedLength(e.Filename); !known {
			ps = append(ps, e.Filename)
		}
	}
	if len(ps) > 0 {
		queueLengths(ps, front)
	}
}

// lengthWorker reads queued paths one at a time, waking the interface a few times a second at most.
func lengthWorker() {
	last := time.Time{}
	for range lengths.kick {
		for {
			lengths.Lock()
			if len(lengths.q) == 0 {
				lengths.Unlock()
				break
			}
			p := lengths.q[0]
			lengths.q = lengths.q[1:]
			delete(lengths.queued, p)
			lengths.Unlock()
			readLength(p)
			lengths.Lock()
			done, wake := len(lengths.q) == 0, lengths.wake
			lengths.Unlock()
			if wake != nil && (done || time.Since(last) > 200*time.Millisecond) {
				wake()
				last = time.Now()
			}
		}
	}
}

// readLength refreshes one cache entry, keeping it when the file is unchanged.
func readLength(p string) {
	var size int64
	var mtime time.Time
	if st, err := os.Stat(p); err == nil {
		size, mtime = st.Size(), st.ModTime()
	}
	lengths.Lock()
	e, had := lengths.m[p]
	lengths.Unlock()
	if had && e.size == size && e.mtime.Equal(mtime) {
		return
	}
	sec, ok := trackLength(p)
	lengths.Lock()
	if cur := lengths.m[p]; !(cur.exact && cur.size == size && cur.mtime.Equal(mtime)) { // mpv may have reported it meanwhile
		lengths.m[p] = lengthEntry{size, mtime, sec, ok, false}
	}
	lengths.Unlock()
}

// totalLength is the playlist's summed known lengths, with "+" while some are unknown; "" when empty.
func totalLength(list []entry) string {
	if len(list) == 0 {
		return ""
	}
	sum, more := 0.0, false
	for _, e := range list {
		if sec, ok, _ := cachedLength(e.Filename); ok {
			sum += sec
		} else {
			more = true
		}
	}
	s := fmtLong(sum)
	if more {
		s += "+"
	}
	return s
}

// fmtLong is fmtTime from one hour on as h:mm:ss.
func fmtLong(sec float64) string {
	t := int(sec)
	if t < 3600 {
		return fmtTime(sec)
	}
	return fmt.Sprintf("%d:%02d:%02d", t/3600, t/60%60, t%60)
}

// rowLength is the length shown on a playlist row: mpv's for the playing entry, else the cache's.
func rowLength(e entry, s state) string {
	if e.Current && s.Dur > 0 {
		return fmtTime(s.Dur)
	}
	if sec, ok, _ := cachedLength(e.Filename); ok {
		return fmtTime(sec)
	}
	return ""
}
