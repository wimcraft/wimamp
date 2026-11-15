package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func be32(v uint32) []byte    { return binary.BigEndian.AppendUint32(nil, v) }
func le32(v uint32) []byte    { return binary.LittleEndian.AppendUint32(nil, v) }
func cat(bs ...[]byte) []byte { return bytes.Join(bs, nil) }

func flacFixture(rate, samples uint64) []byte {
	si := make([]byte, 34)
	si[10], si[11], si[12] = byte(rate>>12), byte(rate>>4), byte(rate<<4)
	si[13] = byte(samples >> 32 & 0x0f)
	binary.BigEndian.PutUint32(si[14:], uint32(samples))
	return cat([]byte("fLaC"), []byte{0x80, 0, 0, 34}, si)
}

func wavFixture(byteRate, data uint32) []byte {
	fmtc := make([]byte, 16)
	binary.LittleEndian.PutUint32(fmtc[8:], byteRate)
	return cat([]byte("RIFF"), le32(0), []byte("WAVE"), []byte("LIST"), le32(3), []byte("abc\x00"),
		[]byte("fmt "), le32(16), fmtc, []byte("data"), le32(data), make([]byte, data))
}

func aiffFixture(frames uint32) []byte {
	comm := cat([]byte{0, 2}, be32(frames), []byte{0, 16}, []byte{0x40, 0x0e, 0xac, 0x44, 0, 0, 0, 0, 0, 0}) // 44100 Hz
	return cat([]byte("FORM"), be32(0), []byte("AIFF"), []byte("COMM"), be32(18), comm)
}

func oggPage(granule int64, packet []byte) []byte {
	h := make([]byte, 27)
	copy(h, "OggS")
	binary.LittleEndian.PutUint64(h[6:], uint64(granule))
	h[26] = 1
	return cat(h, []byte{byte(len(packet))}, packet)
}

func mp4Box(typ string, body ...[]byte) []byte {
	b := cat(body...)
	return cat(be32(uint32(8+len(b))), []byte(typ), b)
}

func mvhd(version byte, scale uint32, dur uint64) []byte {
	if version == 1 {
		return mp4Box("mvhd", []byte{1, 0, 0, 0}, make([]byte, 16), be32(scale), binary.BigEndian.AppendUint64(nil, dur), make([]byte, 80))
	}
	return mp4Box("mvhd", []byte{0, 0, 0, 0}, make([]byte, 8), be32(scale), be32(uint32(dur)), make([]byte, 80))
}

var mp3Frame = []byte{0xff, 0xfb, 0x90, 0x00} // MPEG1 Layer III, 128 kbps, 44100 Hz, stereo

func lengthFixtures() []struct {
	name string
	read func(r *bytes.Reader) (float64, bool)
	data []byte
	want float64
} {
	vorbisID := cat([]byte("\x01vorbis"), le32(0), []byte{2}, le32(44100), make([]byte, 15))
	opusID := cat([]byte("OpusHead"), []byte{1, 2, 0x38, 0x01}, le32(48000), make([]byte, 3)) // pre-skip 312
	id3 := cat([]byte("ID3"), []byte{3, 0, 0, 0, 0, 0, 10}, make([]byte, 10))
	tag := cat([]byte("TAG"), make([]byte, 125))
	type fx = struct {
		name string
		read func(r *bytes.Reader) (float64, bool)
		data []byte
		want float64
	}
	w := func(f func(io.ReaderAt, int64) (float64, bool)) func(*bytes.Reader) (float64, bool) {
		return func(b *bytes.Reader) (float64, bool) { return f(b, b.Size()) }
	}
	return []fx{
		{"flac", w(flacLength), flacFixture(44100, 44100*242), 242},
		{"wav", w(wavLength), wavFixture(100, 300), 3},
		{"aiff", w(aiffLength), aiffFixture(44100 * 195), 195},
		{"vorbis", w(oggLength), cat(oggPage(0, vorbisID), make([]byte, 100), oggPage(44100*60, []byte{0})), 60},
		{"opus", w(oggLength), cat(oggPage(0, opusID), oggPage(48000*10+312, []byte{0})), 10},
		{"mp4 v0, moov last", w(mp4Length), cat(mp4Box("ftyp", []byte("M4A ")), mp4Box("mdat", make([]byte, 50)), mp4Box("moov", mvhd(0, 1000, 195000))), 195},
		{"mp4 v1, moov first", w(mp4Length), cat(mp4Box("ftyp", []byte("M4A ")), mp4Box("moov", mp4Box("udta"), mvhd(1, 44100, 44100*90))), 90},
		{"mp3 xing", w(mp3Length), cat(id3, mp3Frame, make([]byte, 32), []byte("Xing"), be32(1), be32(1000), make([]byte, 400)), 1000 * 1152 / 44100.0},
		{"mp3 vbri", w(mp3Length), cat(mp3Frame, make([]byte, 32), []byte("VBRI"), make([]byte, 10), be32(500), make([]byte, 400)), 500 * 1152 / 44100.0},
		{"mp3 cbr", w(mp3Length), cat(id3, mp3Frame, make([]byte, 16000-4), tag), 1},
	}
}

func TestLengthReaders(t *testing.T) {
	for _, f := range lengthFixtures() {
		got, ok := f.read(bytes.NewReader(f.data))
		if !ok || math.Abs(got-f.want) > 0.01 {
			t.Errorf("%s: got %v %v, want %v", f.name, got, ok, f.want)
		}
		if _, ok := f.read(bytes.NewReader(f.data[:8])); ok {
			t.Errorf("%s: truncated file has a length", f.name)
		}
		if _, ok := f.read(bytes.NewReader(make([]byte, 4096))); ok {
			t.Errorf("%s: zeros have a length", f.name)
		}
	}
}

func FuzzLengthReaders(f *testing.F) {
	for _, fx := range lengthFixtures() {
		f.Add(fx.data)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		for _, read := range lengthReaders {
			read(bytes.NewReader(b), int64(len(b))) // must not panic or hang
		}
	})
}

func TestTrackLengthFiles(t *testing.T) {
	dir := t.TempDir()
	wav := filepath.Join(dir, "a.WAV")
	os.WriteFile(wav, wavFixture(100, 200), 0o644)
	os.WriteFile(filepath.Join(dir, "b.wma"), wavFixture(100, 200), 0o644)
	if sec, ok := trackLength(wav); !ok || sec != 2 {
		t.Errorf("wav: %v %v", sec, ok)
	}
	for _, p := range []string{filepath.Join(dir, "b.wma"), filepath.Join(dir, "gone.mp3"), "https://example.com/x.mp3"} {
		if _, ok := trackLength(p); ok {
			t.Errorf("%s has a length", p)
		}
	}
}

// resetLengths empties the cache and keeps the background worker from starting, so tests drive it by hand.
func resetLengths() {
	lengths.Lock()
	lengths.m, lengths.q, lengths.queued = map[string]lengthEntry{}, nil, map[string]bool{}
	lengths.Unlock()
	lengths.once.Do(func() {})
}

func TestLengthQueue(t *testing.T) {
	resetLengths()
	queueLengths([]string{"a", "b", "a"}, false)
	queueLengths([]string{"b", "c"}, true)
	if got := lengths.q; len(got) != 3 || got[0] != "b" || got[1] != "c" || got[2] != "a" {
		t.Errorf("queue %v, want [b c a]", got)
	}
}

func TestLengthCache(t *testing.T) {
	resetLengths()
	p := filepath.Join(t.TempDir(), "a.wav")
	os.WriteFile(p, wavFixture(100, 200), 0o644)
	readLength(p)
	if sec, ok, _ := cachedLength(p); !ok || sec != 2 {
		t.Fatalf("read: %v %v", sec, ok)
	}
	learnLength(p, 2.5) // mpv knows better
	readLength(p)
	if sec, _, _ := cachedLength(p); sec != 2.5 {
		t.Errorf("header read replaced mpv's length: %v", sec)
	}
	os.WriteFile(p, wavFixture(100, 400), 0o644) // edited: re-read
	os.Chtimes(p, time.Now(), time.Now().Add(time.Minute))
	readLength(p)
	if sec, _, _ := cachedLength(p); sec != 4 {
		t.Errorf("edited file not re-read: %v", sec)
	}
	os.Remove(p)
	readLength(p)
	if _, ok, known := cachedLength(p); ok || !known {
		t.Errorf("deleted file: ok %v known %v", ok, known)
	}
}

func TestTotalLength(t *testing.T) {
	resetLengths()
	set := func(p string, sec float64) { lengths.m[p] = lengthEntry{sec: sec, ok: true} }
	set("a", 60)
	set("b", 150)
	set("c", 45)
	set("long", 3725)
	list := func(ps ...string) (l []entry) {
		for _, p := range ps {
			l = append(l, entry{Filename: p})
		}
		return
	}
	for _, c := range []struct {
		list []entry
		want string
	}{
		{nil, ""},
		{list("a", "b", "c"), "4:15"},
		{list("long", "x.wma"), "1:02:05+"},
	} {
		if got := totalLength(c.list); got != c.want {
			t.Errorf("total %v = %q, want %q", c.list, got, c.want)
		}
	}
}
