package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/ncruces/zenity"
	"golang.org/x/image/font/gofont/goregular"
)

// Size in points: 1 logical pixel per point, which Retina still draws crisply at 2x.
const (
	gw, gh = 300, 374
	scale  = 1
	rowH   = 15
	rows   = 13
)

var (
	fontSource = func() *text.GoTextFaceSource {
		source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
		if err != nil {
			panic(err) // embedded font data is fixed at build time
		}
		return source
	}()
	face      = &text.GoTextFace{Source: fontSource, Size: 11}
	smallFace = &text.GoTextFace{Source: fontSource, Size: 8}
	cPanel    = color.RGBA{0x2b, 0x2c, 0x40, 0xff}
	cBar      = color.RGBA{0x22, 0x22, 0x33, 0xff}
	cDark     = color.RGBA{0x10, 0x10, 0x18, 0xff}
	cBlack    = color.RGBA{0x00, 0x00, 0x00, 0xff}
	cGreen    = color.RGBA{0x70, 0xe0, 0x45, 0xff}
	cDim      = color.RGBA{0x35, 0x70, 0x30, 0xff}
	cWhite    = color.RGBA{0xee, 0xf1, 0xf4, 0xff}
	cSel      = color.RGBA{0x00, 0x19, 0x9c, 0xff}
	cGold     = color.RGBA{0xc2, 0xb4, 0x80, 0xff}
	cErr      = color.RGBA{0xff, 0x60, 0x40, 0xff}
	cBtn      = color.RGBA{0xa7, 0xb7, 0xc1, 0xff} // button face
	cBtnHi    = color.RGBA{0xe1, 0xed, 0xf2, 0xff}
	cBtnLo    = color.RGBA{0x42, 0x4a, 0x5b, 0xff}
	cInk      = color.RGBA{0x29, 0x3b, 0x49, 0xff} // icons and text on buttons
)

var (
	rTitle   = image.Rect(0, 0, gw, 14)
	rClose   = image.Rect(gw-13, 2, gw-3, 12)
	rPin     = image.Rect(3, 2, 13, 12) // always on top
	rLCD     = image.Rect(10, 22, 102, 64)
	rMarquee = image.Rect(106, 23, gw-8, 40)
	rVol     = image.Rect(106, 58, 226, 66)
	rShuf    = image.Rect(232, 52, 261, 69)
	rRep     = image.Rect(263, 52, 292, 69)
	rSeek    = image.Rect(16, 72, gw-10, 83)
	rPLTitle = image.Rect(0, 116, gw, 130)
	rList    = image.Rect(10, 136, gw-17, 136+rows*rowH)
	rScroll  = image.Rect(gw-14, 136, gw-7, 136+rows*rowH)
	rAdd     = image.Rect(10, gh-29, 35, gh-11)
	rRem     = image.Rect(39, gh-29, 64, gh-11)
	rAddFile = image.Rect(10, gh-65, 50, gh-47)
	rAddDir  = image.Rect(10, gh-47, 50, gh-29)
)

// transport buttons, left to right, and the action each one runs
var transportActions = []string{"prev", "resume", "toggle", "stop", "next"}

func transport(i int) image.Rectangle { return image.Rect(16+i*25, 88, 40+i*25, 108) }

type gui struct {
	mu    sync.Mutex
	s     state
	err   error
	flash string
	dirty bool

	keys        map[string]string
	pending     string
	cursor, top int
	searchTop   int
	synced      bool

	drag         string          // "win", "seek", "vol" while the left button is held
	down         image.Rectangle // the button held under the mouse, drawn pressed until release
	volUntil     time.Time       // the marquee shows the volume instead of the title until then
	dragX, dragY int
	seekFrac     float64
	lastClick    time.Time
	lastRow      int
	menu         bool // ADD's FILE/DIR menu
	help         bool
	search       search
	dialog       atomic.Bool
	wasFocused   bool
	quit         bool
	resident     bool // closing hides the window and the process waits for the next `wimamp gui`
	hidden       atomic.Bool
	show         atomic.Bool // set by the control socket
	scrolls      bool
	started      bool
	step         int64
}

func runGUI() error {
	if err := ensureDaemon(); err != nil {
		return err
	}
	// A resident window from an earlier `wimamp gui` shows in a few ms; a new process needs ~0.5 s.
	if c, err := dialGUI(); err == nil {
		_, err = c.Write([]byte("show\n"))
		c.Close()
		return err
	}
	unlock, err := lock("gui", false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wimamp: the window is already starting")
		return nil
	}
	defer unlock()
	keys, err := loadKeys()
	if err != nil {
		return err
	}
	g := &gui{keys: keys, lastRow: -1}
	if ln, err := listenGUI(); err == nil { // without it the window simply exits on close
		defer ln.Close()
		g.resident = true
		go g.serve(ln)
	}
	g.exec(nil)
	go func() {
		for range time.Tick(250 * time.Millisecond) {
			if !g.hidden.Load() {
				select { // also keeps focus checks and the marquee ticking; skip a tick if the queue is busy
				case queue <- func() { g.exec(nil) }:
				default:
				}
			}
		}
	}()
	ebiten.SetWindowTitle("Wimamp")
	ebiten.SetWindowSize(gw*scale, gh*scale)
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(true)
	ebiten.SetScreenClearedEveryFrame(false) // Draw repaints only when dirty
	// frames run only on input or ScheduleFrame (from the poller); the non-deprecated
	// vsync/TPS settings still idle at ~5% CPU because every frame is presented.
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMinimum) //nolint:staticcheck
	// single thread: Update runs on the main thread, where the AppKit calls in activate() must happen
	return ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{SingleThread: true})
}

// serve takes "show" and "quit" from later `wimamp gui` / `wimamp quit` calls.
func (g *gui) serve(ln net.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go g.handle(c) // one goroutine per client: a silent one can't delay the next
	}
}

func (g *gui) handle(c net.Conn) {
	defer c.Close()
	c.SetReadDeadline(time.Now().Add(time.Second))
	b := make([]byte, 16)
	n, _ := c.Read(b)
	switch strings.TrimSpace(string(b[:n])) {
	case "show":
		g.show.Store(true)
	case "quit":
		g.mu.Lock()
		g.quit, g.resident = true, false
		g.mu.Unlock()
	}
	ebiten.ScheduleFrame() //nolint:staticcheck
}

// closeWin hides a resident window (resetting transient UI state) or ends a plain one.
func (g *gui) closeWin() error {
	if !g.resident {
		return ebiten.Termination
	}
	ebiten.SetWindowVisible(false)
	g.hidden.Store(true)
	g.quit, g.wasFocused, g.help, g.menu, g.drag, g.search, g.pending = false, false, false, false, "", search{}, ""
	return nil
}

func (g *gui) reveal() {
	g.hidden.Store(false)
	g.synced, g.dirty = false, true // the cursor jumps to the playing track again
	ebiten.SetWindowVisible(true)
	activate()
	g.do(nil)
}

// do queues a player call (nil just polls) behind earlier ones, then refreshes the state.
// It never blocks: Update holds g.mu, and the queue's worker needs g.mu to store results.
func (g *gui) do(f func() error) {
	select {
	case queue <- func() { g.exec(f) }:
	default:
		g.flash, g.dirty = "player busy, try again", true
	}
}

// exec runs a player call (nil just polls) off the UI thread, then stores fresh state.
func (g *gui) exec(f func() error) {
	var err error
	if f != nil {
		err = f()
	}
	s, ferr := fetch()
	defer ebiten.ScheduleFrame() //nolint:staticcheck
	g.mu.Lock()
	defer g.mu.Unlock()
	if !reflect.DeepEqual(s, g.s) || (ferr == nil) != (g.err == nil) {
		g.dirty = true
	}
	g.s, g.err = s, ferr
	if err != nil {
		g.flash, g.dirty = err.Error(), true
	}
	if !g.synced && ferr == nil {
		g.cursor, g.synced, g.dirty = currentIndex(s.List, 0), true, true
		g.keepVisible()
	}
}

func (g *gui) Layout(int, int) (int, int) { return gw, gh }

func (g *gui) Update() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.started { // after GLFW set up the app: no Dock icon, and take focus
		g.started = true
		accessory()
		activate()
	}
	if g.show.Swap(false) {
		g.reveal()
	}
	if g.quit { // before the hidden check: `wimamp quit` must also end a hidden window
		return g.closeWin()
	}
	if g.hidden.Load() {
		return nil
	}
	// Spotlight-style: close when focus moves elsewhere, but not to our own file dialog
	f := ebiten.IsFocused()
	switch {
	case g.dialog.Load():
		g.wasFocused = false
	case g.wasFocused && !f && !ebiten.IsWindowFloating(): // pinned on top: stay open
		return g.closeWin()
	default:
		g.wasFocused = f
	}
	for _, k := range pressedKeys() {
		g.flash, g.dirty = "", true
		switch {
		case g.help: // any key closes the help
			g.help = false
		case g.search.on:
			if i := g.search.key(k, g.s.List); i >= 0 {
				g.cursor = i
				g.do(playerAction("play", i, len(g.s.List)))
			}
		case k == "esc":
			return g.closeWin()
		default:
			g.act(resolve(g.keys, &g.pending, k))
		}
		g.keepVisible()
	}
	g.mouse()
	g.clamp()
	if !g.volUntil.IsZero() && time.Now().After(g.volUntil) {
		g.volUntil, g.dirty = time.Time{}, true
	}
	if step := time.Now().UnixMilli() / 200; g.scrolls && step != g.step {
		g.step, g.dirty = step, true
	}
	return nil
}

var namedKeys = map[ebiten.Key]string{
	ebiten.KeyEnter: "enter", ebiten.KeyNumpadEnter: "enter", ebiten.KeyTab: "tab",
	ebiten.KeyArrowUp: "up", ebiten.KeyArrowDown: "down", ebiten.KeyArrowLeft: "left", ebiten.KeyArrowRight: "right",
	ebiten.KeyHome: "home", ebiten.KeyEnd: "end", ebiten.KeyPageUp: "pgup", ebiten.KeyPageDown: "pgdown",
	ebiten.KeyBackspace: "backspace", ebiten.KeyDelete: "delete", ebiten.KeyEscape: "esc",
}

// terminals send these control keys as enter/tab/backspace/esc; do the same so muscle memory carries over
var ctrlAlias = map[string]string{"ctrl+m": "enter", "ctrl+i": "tab", "ctrl+h": "backspace", "ctrl+[": "esc"}

// pressedKeys turns this frame's input into keys.conf names ("j", "G", "enter", "ctrl+d", "space").
func pressedKeys() []string {
	var out []string
	for k, name := range namedKeys {
		if d := inpututil.KeyPressDuration(k); d == 1 || d > 20 && d%4 == 0 { // with key repeat
			out = append(out, name)
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
			if inpututil.IsKeyJustPressed(k) {
				out = append(out, "ctrl+"+strings.ToLower(k.String()))
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) {
			out = append(out, "ctrl+[")
		}
		for i, k := range out {
			if a, ok := ctrlAlias[k]; ok {
				out[i] = a
			}
		}
		return out
	}
	for _, r := range ebiten.AppendInputChars(nil) {
		if r == ' ' {
			out = append(out, "space")
		} else {
			out = append(out, string(r))
		}
	}
	return out
}

func (g *gui) act(a string) {
	if c, ok := moveCursor(a, g.cursor, rows, g.s.List); ok {
		g.cursor = c
		return
	}
	switch a {
	case "quit":
		g.quit = true
	case "quit-player":
		g.do(func() error { _, err := call(simple["quit"]); return err })
		g.quit = true
	case "search":
		g.search = search{on: true}
	case "help":
		g.help = true
	case "add-files", "add-folder":
		g.pick(a == "add-folder")
	default:
		if f := playerAction(a, g.cursor, len(g.s.List)); f != nil {
			g.do(f)
		}
		if a == "volume-up" || a == "volume-down" {
			g.showVol()
		}
	}
}

// showVol puts the volume in the marquee for a second, then wakes Update to bring the title back.
func (g *gui) showVol() {
	g.volUntil = time.Now().Add(time.Second)
	time.AfterFunc(time.Second+50*time.Millisecond, func() { ebiten.ScheduleFrame() }) //nolint:staticcheck
}

func (g *gui) pick(dir bool) {
	g.menu = false
	if !g.dialog.CompareAndSwap(false, true) { // one picker at a time, or the focus guard breaks
		return
	}
	go func() {
		defer g.dialog.Store(false)
		var paths []string
		var err error
		if dir {
			var p string
			p, err = zenity.SelectFile(zenity.Directory(), zenity.Title("Add folder"))
			paths = []string{p}
		} else {
			paths, err = zenity.SelectFileMultiple(zenity.Title("Add files"))
		}
		if errors.Is(err, zenity.ErrCanceled) {
			return
		}
		queue <- func() { // blocking is fine here: this goroutine holds no lock
			g.exec(func() error {
				if err != nil {
					return err
				}
				return load(paths, "append")
			})
		}
	}()
}

func frac(x int, r image.Rectangle) float64 {
	return max(0, min(1, float64(x-r.Min.X)/float64(r.Dx())))
}

func (g *gui) mouse() {
	x, y := ebiten.CursorPosition()
	p := image.Pt(x, y)
	if _, wy := ebiten.Wheel(); wy != 0 && p.In(rList) {
		if g.search.on {
			g.search.sel = max(min(g.search.sel-int(wy*3), len(match(g.s.List, g.search.q))-1), 0)
		} else {
			g.cursor -= int(wy * 3)
		}
		g.keepVisible()
		g.dirty = true
	}
	if !g.down.Empty() && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.down, g.dirty = image.Rectangle{}, true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.dirty = true
		menu := g.menu
		g.menu = false
		for _, r := range []image.Rectangle{rClose, rPin, rShuf, rRep, rAdd, rRem} {
			if p.In(r) {
				g.down = r
			}
		}
		if menu {
			for _, r := range []image.Rectangle{rAddFile, rAddDir} {
				if p.In(r) {
					g.down = r
				}
			}
		}
		switch {
		case menu && p.In(rAddFile):
			g.pick(false)
		case menu && p.In(rAddDir):
			g.pick(true)
		case p.In(rClose):
			g.quit = true
		case p.In(rPin):
			ebiten.SetWindowFloating(!ebiten.IsWindowFloating())
		case p.In(rTitle) || p.In(rPLTitle):
			g.drag, g.dragX, g.dragY = "win", x, y
		case p.In(rSeek.Inset(-3)):
			g.drag, g.seekFrac = "seek", frac(x, rSeek)
		case p.In(rVol.Inset(-3)):
			g.drag = "vol"
		case p.In(rShuf):
			g.do(toggleShuffle)
		case p.In(rRep):
			g.do(toggleRepeat)
		case p.In(rAdd):
			g.menu = !menu
		case p.In(rRem):
			g.act("remove")
		case p.In(rScroll):
			idx, _, top := g.view()
			thumb := scrollThumb(len(idx), top)
			g.drag, g.dragY = "scroll", thumb.Dy()/2
			if p.In(thumb) {
				g.dragY = y - thumb.Min.Y
			}
		case p.In(rList):
			idx, _, top := g.view()
			if n := top + (y-rList.Min.Y)/rowH; n < len(idx) {
				double := idx[n] == g.lastRow && time.Since(g.lastClick) < 400*time.Millisecond
				g.lastRow, g.lastClick = idx[n], time.Now()
				if g.search.on {
					g.search.sel = n
				} else {
					g.cursor = idx[n]
				}
				if double {
					g.search.on, g.cursor = false, idx[n]
					g.act("play")
				}
			}
		default:
			for i, a := range transportActions {
				if p.In(transport(i)) {
					g.down = transport(i)
					g.act(a)
				}
			}
		}
	}
	if g.drag == "" {
		return
	}
	g.dirty = true
	switch g.drag {
	case "win": // the cursor is window-relative, so moving the window by the delta keeps it under the cursor
		wx, wy := ebiten.WindowPosition()
		ebiten.SetWindowPosition(wx+(x-g.dragX)*scale, wy+(y-g.dragY)*scale)
	case "vol":
		if v := float64(int(frac(x, rVol) * 100)); v != g.s.Vol {
			g.s.Vol = v
			g.do(func() error { _, err := call([]any{"set_property", "volume", v}); return err })
		}
		g.showVol()
	case "seek":
		g.seekFrac = frac(x, rSeek)
	case "scroll":
		g.scrollTo(y - g.dragY)
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if g.drag == "seek" {
			pct := g.seekFrac * 100
			g.do(func() error { _, err := call([]any{"seek", pct, "absolute-percent"}); return err })
		}
		g.drag = ""
	}
}

// view is the rows the list shows: the whole playlist, or the matches while searching.
func (g *gui) view() (idx []int, sel, top int) {
	if g.search.on {
		idx = match(g.s.List, g.search.q)
		return idx, g.search.sel, g.searchTop
	}
	idx = make([]int, len(g.s.List))
	for i := range idx {
		idx[i] = i
	}
	return idx, g.cursor, g.top
}

func (g *gui) clamp() {
	g.cursor = max(min(g.cursor, len(g.s.List)-1), 0)
	g.top = max(0, min(g.top, len(g.s.List)-rows))
	if g.search.on {
		g.searchTop = max(0, min(g.searchTop, len(match(g.s.List, g.search.q))-rows))
	}
}

func (g *gui) keepVisible() {
	g.clamp()
	if g.search.on {
		g.searchTop = max(0, min(g.searchTop, g.search.sel), g.search.sel-rows+1)
	} else {
		g.top = max(0, min(g.top, g.cursor), g.cursor-rows+1)
	}
}

const scrollThumbHeight = 18

func scrollThumb(count, top int) image.Rectangle {
	travel := rScroll.Dy() - scrollThumbHeight
	y := rScroll.Min.Y
	if count > rows {
		y += max(0, min(top, count-rows)) * travel / (count - rows)
	}
	return image.Rect(rScroll.Min.X, y, rScroll.Max.X, y+scrollThumbHeight)
}

func (g *gui) scrollTo(y int) {
	idx, _, _ := g.view()
	travel := rScroll.Dy() - scrollThumbHeight
	top := (max(0, min(y-rScroll.Min.Y, travel))*max(0, len(idx)-rows) + travel/2) / travel
	if g.search.on {
		g.searchTop = top
	} else {
		g.top = top
	}
}

// --- drawing ---

func fill(dst *ebiten.Image, r image.Rectangle, c color.Color) {
	vector.FillRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
}

func label(dst *ebiten.Image, s string, x, y int, c color.Color, size float64) {
	f := face
	if size < 0.75 {
		f = smallFace
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, f, op)
}

func centered(dst *ebiten.Image, s string, r image.Rectangle, c color.Color) {
	label(dst, s, r.Min.X+(r.Dx()-int(text.Advance(s, face)))/2, r.Min.Y+(r.Dy()-13)/2, c, 1)
}

// raised draws a light grey button with a 2px bevel inside a black outline; pressed swaps the bevel.
func raised(dst *ebiten.Image, r image.Rectangle, pressed bool) {
	hi, lo := cBtnHi, cBtnLo
	if pressed {
		hi, lo = lo, hi
	}
	fill(dst, r, cBlack)
	in := r.Inset(1)
	fill(dst, in, lo)
	fill(dst, image.Rect(in.Min.X, in.Min.Y, in.Max.X-1, in.Max.Y-1), hi)
	fill(dst, image.Rect(in.Min.X+1, in.Min.Y+1, in.Max.X-1, in.Max.Y-1), cBtn)
}

// bevel gives every button the same inset highlight and shadow; pressed swaps them.
func bevel(dst *ebiten.Image, r image.Rectangle, pressed bool) {
	hi, lo := color.Color(cWhite), color.Color(cBtnLo)
	if pressed {
		hi, lo = lo, hi
	}
	raised(dst, r, pressed)
	fill(dst, image.Rect(r.Min.X+2, r.Min.Y+2, r.Max.X-2, r.Min.Y+3), hi)
	fill(dst, image.Rect(r.Min.X+2, r.Min.Y+2, r.Min.X+3, r.Max.Y-2), hi)
	fill(dst, image.Rect(r.Min.X+3, r.Max.Y-3, r.Max.X-2, r.Max.Y-2), lo)
	fill(dst, image.Rect(r.Max.X-3, r.Min.Y+3, r.Max.X-2, r.Max.Y-2), lo)
}

func button(dst *ebiten.Image, r image.Rectangle, s string, pressed bool) {
	bevel(dst, r, pressed)
	if pressed { // the face sinks with the bevel
		r = r.Add(image.Pt(1, 1))
	}
	label(dst, s, r.Min.X+(r.Dx()-int(text.Advance(s, smallFace)))/2, r.Min.Y+(r.Dy()-10)/2, cInk, 0.5)
}

// LCD digits: 7 segments, a-g clockwise from the top, then the middle one.
const lcdW, lcdH, lcdT = 10, 18, 2

var lcdSegs = map[rune]string{
	'0': "abcdef", '1': "bc", '2': "abged", '3': "abgcd", '4': "fgbc",
	'5': "afgcd", '6': "afgedc", '7': "abc", '8': "abcdefg", '9': "abcdfg", '-': "g",
}

func lcdAdvance(r rune) int {
	if r == ':' {
		return lcdT + 4
	}
	return lcdW + 3
}

func lcdWidth(s string) (w int) {
	for _, r := range s {
		w += lcdAdvance(r)
	}
	return w
}

// lcdRects lays out s as segment rectangles from (x, y), the font's own colon being two tiny '+'.
func lcdRects(s string, x, y int) (out []image.Rectangle) {
	const w, h, t = lcdW, lcdH, lcdT
	m := y + h/2
	seg := map[rune]image.Rectangle{
		'a': image.Rect(t, 0, w-t, t), 'b': image.Rect(w-t, t, w, h/2-1), 'c': image.Rect(w-t, h/2+1, w, h-t),
		'd': image.Rect(t, h-t, w-t, h), 'e': image.Rect(0, h/2+1, t, h-t), 'f': image.Rect(0, t, t, h/2-1),
		'g': image.Rect(t, h/2-1, w-t, h/2+1),
	}
	for _, r := range s {
		if r == ':' {
			out = append(out, image.Rect(x+1, m-5, x+1+t, m-5+t), image.Rect(x+1, m+3, x+1+t, m+3+t))
		}
		for _, c := range lcdSegs[r] {
			out = append(out, seg[c].Add(image.Pt(x, y)))
		}
		x += lcdAdvance(r)
	}
	return out
}

// sunken draws a groove cut into the panel: shadow on the top and left, light on the bottom and right.
func sunken(dst *ebiten.Image, r image.Rectangle) {
	fill(dst, r, cBtnHi)
	fill(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X-1, r.Max.Y-1), cBlack)
	fill(dst, image.Rect(r.Min.X+1, r.Min.Y+1, r.Max.X-1, r.Max.Y-1), cBtnLo)
	fill(dst, image.Rect(r.Min.X+2, r.Min.Y+2, r.Max.X-1, r.Max.Y-1), cDark)
}

// grip draws a slider handle: a grey button with three grip lines in the middle.
func grip(dst *ebiten.Image, r image.Rectangle) {
	raised(dst, r, false)
	mx := r.Min.X + r.Dx()/2
	for i := -2; i <= 2; i += 2 {
		fill(dst, image.Rect(mx+i, r.Min.Y+3, mx+i+1, r.Max.Y-3), cInk)
	}
}

// volStops color the volume track by level: green at 0, yellow at 50, red at 100.
var volStops = []color.RGBA{
	{0x3c, 0x8c, 0x28, 0xff}, {0x8c, 0xd2, 0x3c, 0xff}, {0xdc, 0xdc, 0x32, 0xff}, {0xe6, 0xa0, 0x32, 0xff}, {0xd2, 0x3c, 0x28, 0xff},
}

func volColor(v float64) color.RGBA {
	f := max(0, min(v, 100)) / 100 * float64(len(volStops)-1)
	i := min(int(f), len(volStops)-2)
	return lerp(volStops[i], volStops[i+1], f-float64(i))
}

// lerp mixes a into b by f (0..1).
func lerp(a, b color.RGBA, f float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*f) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 0xff}
}

// fit cuts s to at most w pixels.
func fit(s string, w int) string {
	if int(text.Advance(s, face)) <= w {
		return s
	}
	rs := []rune(s)
	for len(rs) > 0 && int(text.Advance(string(rs)+"..", face)) > w {
		rs = rs[:len(rs)-1]
	}
	return string(rs) + ".."
}

func marqueeArea() image.Rectangle {
	return image.Rect(rMarquee.Min.X+3, rMarquee.Min.Y+1, rMarquee.Max.X-3, rMarquee.Max.Y-1)
}

// tri draws a pixel-art triangle h pixels tall pointing right (or left).
func tri(dst *ebiten.Image, x, y, h int, right bool, c color.Color) {
	for i := 0; i <= h/2; i++ {
		col := x + i
		if !right {
			col = x + h/2 - i
		}
		fill(dst, image.Rect(col, y+i, col+1, y+h-i), c)
	}
}

func icon(dst *ebiten.Image, a string, r image.Rectangle, c color.Color) {
	x, y := r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2-4
	bar := func(x int) { fill(dst, image.Rect(x, y, x+2, y+9), c) }
	switch a {
	case "prev":
		bar(x - 4)
		tri(dst, x-2, y, 8, false, c)
	case "resume", "playing":
		tri(dst, x-2, y, 8, true, c)
	case "toggle", "paused":
		bar(x - 3)
		bar(x + 1)
	case "stop", "stopped":
		fill(dst, image.Rect(x-4, y, x+4, y+8), c)
	case "next":
		tri(dst, x-3, y, 8, true, c)
		bar(x + 2)
	}
}

// Pixel art for the mode toggles, drawn 1:1 like the transport icons so they stay crisp.
var (
	shuffleArt = []string{
		".........#.",
		"##....#####",
		"##....#####",
		"..##.##..#.",
		"...###.....",
		"...###.....",
		"..##.##..#.",
		"##....#####",
		"##....#####",
		".........#.",
	}
	repeatArt = []string{
		".......#...",
		".########..",
		"#########..",
		"##.....#...",
		"##.........",
		"##.........",
		"##.........",
		"##########.",
		".#########.",
	}
)

func modeIcon(dst *ebiten.Image, r image.Rectangle, shuffle bool, c color.Color) {
	art := repeatArt
	if shuffle {
		art = shuffleArt
	}
	x := r.Min.X + (r.Dx()-len(art[0]))/2 + 4 // past the LED
	y := r.Min.Y + (r.Dy()-len(art))/2
	for j, row := range art {
		for i, px := range row {
			if px == '#' {
				fill(dst, image.Rect(x+i, y+j, x+i+1, y+j+1), c)
			}
		}
	}
}

func titleBar(dst *ebiten.Image, r image.Rectangle, s string) {
	fill(dst, r, cBar)
	fill(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1), cBtnLo)
	fill(dst, image.Rect(r.Min.X, r.Max.Y-2, r.Max.X, r.Max.Y-1), cBlack)
	w := int(text.Advance(s, face))
	mid := r.Min.Y + r.Dy()/2
	for _, y := range []int{mid - 3, mid, mid + 3} {
		fill(dst, image.Rect(r.Min.X+16, y, (r.Dx()-w)/2-4, y+1), cGold)
		fill(dst, image.Rect((r.Dx()+w)/2+4, y, r.Max.X-16, y+1), cGold)
	}
	label(dst, s, r.Min.X+(r.Dx()-w)/2, r.Min.Y+1, cWhite, 1)
}

// frame gives each pane an independent dark outline and inset highlight.
func frame(dst *ebiten.Image, r image.Rectangle) {
	fill(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1), cBlack)
	fill(dst, image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y), cBlack)
	fill(dst, image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y), cBlack)
	fill(dst, image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y), cBlack)
	fill(dst, image.Rect(r.Min.X+2, r.Max.Y-3, r.Max.X-2, r.Max.Y-2), cBtnLo)
	fill(dst, image.Rect(r.Min.X+2, r.Min.Y+2, r.Min.X+3, r.Max.Y-2), cBtnLo)
	fill(dst, image.Rect(r.Max.X-3, r.Min.Y+2, r.Max.X-2, r.Max.Y-2), cBtnLo)
}

func (g *gui) Draw(screen *ebiten.Image) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.dirty {
		return
	}
	g.dirty = false
	s := g.s
	screen.Fill(cPanel)
	// The two sections have the recessed, double-edged frame of the classic skin.
	fill(screen, image.Rect(0, 14, gw, 15), cBtnLo)
	fill(screen, image.Rect(0, 114, gw, 116), cBlack)
	fill(screen, image.Rect(0, 130, gw, 131), cBtnLo)
	sunken(screen, image.Rect(rList.Min.X-2, rList.Min.Y-2, rList.Max.X+2, rList.Max.Y+2))
	fill(screen, image.Rect(0, gh-2, gw, gh-1), cBtnLo)

	titleBar(screen, rTitle, "WIMAMP")
	button(screen, rClose, "x", g.down == rClose)
	pinned := ebiten.IsWindowFloating()
	raised(screen, rPin, g.down == rPin)
	led := cDim
	if pinned {
		led = cGreen
	}
	fill(screen, image.Rect(rPin.Min.X+3, rPin.Min.Y+3, rPin.Max.X-3, rPin.Max.Y-3), led)

	// LCD: state icon and elapsed time
	fill(screen, rLCD, cBlack)
	for y := rLCD.Min.Y + 2; y < rLCD.Max.Y-1; y += 2 {
		for x := rLCD.Min.X + 2; x < rLCD.Max.X-1; x += 2 {
			fill(screen, image.Rect(x, y, x+1, y+1), cBar)
		}
	}
	st, t := "playing", fmtTime(s.Pos)
	switch {
	case g.err != nil || s.Idle:
		st, t = "stopped", "-:--"
	case s.Paused:
		st = "paused"
	}
	icon(screen, st, image.Rect(rLCD.Min.X+2, rLCD.Min.Y+4, rLCD.Min.X+14, rLCD.Min.Y+16), cGreen)
	for _, r := range lcdRects(t, rLCD.Max.X-6-lcdWidth(t), rLCD.Min.Y+12) {
		fill(screen, r, cGreen)
	}

	// marquee: "N. Title (m:ss)", scrolling when it doesn't fit
	fill(screen, rMarquee, cBlack)
	msg, c := "stopped", cGreen
	if i := currentIndex(s.List, -1); g.err != nil {
		msg, c = "player not running", cErr
	} else if !s.Idle && i >= 0 {
		msg = fmt.Sprintf("%d. %s (%s)", i+1, s.Title, fmtTime(s.Dur))
	}
	if !g.volUntil.IsZero() {
		msg, c = fmt.Sprintf("VOLUME: %d%%", int(s.Vol)), cGreen
	}
	area := marqueeArea()
	g.scrolls = int(text.Advance(msg, face)) > area.Dx()
	if g.scrolls {
		rs := []rune(msg + "  ***  ")
		off := int(g.step % int64(len(rs)))
		msg = string(rs[off:]) + string(rs[:off])
	}
	label(screen.SubImage(area).(*ebiten.Image), msg, area.Min.X, area.Min.Y, c, 0.75)

	// Only show values actually provided by the player; no decorative bitrate or stereo indicators.
	label(screen, fmt.Sprintf("VOL %d%%", int(s.Vol)), 106, 43, cGreen, 0.6)
	if s.Dur > 0 {
		label(screen, fmtTime(s.Dur), 169, 43, cGreen, 0.6)
	}

	// volume, shuffle, repeat
	track, vc := rVol.Inset(2), volColor(s.Vol)
	fill(screen, image.Rect(track.Min.X, track.Min.Y-1, track.Max.X, track.Max.Y+1), lerp(cPanel, vc, 0.25))
	fill(screen, track, lerp(vc, cPanel, 0.25))
	fill(screen, image.Rect(track.Min.X, track.Min.Y+1, track.Max.X, track.Max.Y-1), lerp(vc, cWhite, 0.18))
	const kw = 14
	kx := rVol.Min.X + int(min(s.Vol, 100)/100*float64(rVol.Dx()-kw))
	grip(screen, image.Rect(kx, rVol.Min.Y-2, kx+kw, rVol.Max.Y+2))
	for _, t := range []struct {
		r       image.Rectangle
		shuffle bool
		on      bool
	}{{rShuf, true, s.Shuffle}, {rRep, false, s.Repeat}} {
		pressed := g.down == t.r
		bevel(screen, t.r, pressed)
		r := t.r
		if pressed {
			r = r.Add(image.Pt(1, 1))
		}
		led := cDim
		if t.on {
			led = cGreen
		}
		my := r.Min.Y + r.Dy()/2
		fill(screen, image.Rect(r.Min.X+5, my-2, r.Min.X+9, my+2), led)
		modeIcon(screen, r, t.shuffle, cInk)
	}

	// seek bar
	fill(screen, rSeek, cBtnLo)
	fill(screen, image.Rect(rSeek.Min.X+1, rSeek.Min.Y+1, rSeek.Max.X-1, rSeek.Max.Y-1), cBar)
	fill(screen, image.Rect(rSeek.Min.X+1, rSeek.Min.Y+1, rSeek.Max.X-1, rSeek.Min.Y+2), cDark)
	f := 0.0
	if g.drag == "seek" {
		f = g.seekFrac
	} else if s.Dur > 0 {
		f = s.Pos / s.Dur
	}
	const sw = 28 // wider than the volume knob, as in the original
	sx := rSeek.Min.X + int(f*float64(rSeek.Dx()-sw))
	seekKnob := image.Rect(sx, rSeek.Min.Y-2, sx+sw, rSeek.Max.Y+2)
	fill(screen, seekKnob, cBlack)
	fill(screen, seekKnob.Inset(1), cGold)
	fill(screen, image.Rect(sx+2, seekKnob.Min.Y+2, sx+sw-2, seekKnob.Max.Y-2), cBtnLo)
	fill(screen, image.Rect(sx+3, seekKnob.Min.Y+3, sx+sw-3, seekKnob.Max.Y-3), cGold)
	fill(screen, image.Rect(sx+4, seekKnob.Min.Y+3, sx+sw-4, seekKnob.Min.Y+4), cBtnHi)

	for i, a := range transportActions {
		r := transport(i)
		pressed := g.down == r
		bevel(screen, r, pressed)
		if pressed {
			r = r.Add(image.Pt(1, 1))
		}
		icon(screen, a, r.Add(image.Pt(1, 1)), cBtnHi)
		icon(screen, a, r, cInk)
	}

	// playlist
	titleBar(screen, rPLTitle, "WIMAMP PLAYLIST")
	fill(screen, rList, cBlack)
	if len(s.List) == 0 {
		label(screen, "empty - a: add files, ?: keys", rList.Min.X+4, rList.Min.Y, cDim, 0.75)
	}
	idx, sel, top := g.view()
	fill(screen, rScroll, cDark)
	thumb := scrollThumb(len(idx), top)
	raised(screen, thumb, false)
	fill(screen, image.Rect(thumb.Min.X+2, thumb.Min.Y+3, thumb.Max.X-2, thumb.Max.Y-3), cGold)
	for n := top; n < len(idx) && n < top+rows; n++ {
		i := idx[n]
		e := s.List[i]
		y := rList.Min.Y + (n-top)*rowH
		if n == sel {
			fill(screen, image.Rect(rList.Min.X, y, rList.Max.X, y+rowH), cSel)
		}
		c, right := cGreen, ""
		if n == sel {
			c = cWhite
		}
		if e.Current {
			if s.Dur > 0 {
				right = fmtTime(s.Dur)
			}
		}
		rw := int(text.Advance(right, face))
		label(screen, fit(fmt.Sprintf("%d. %s", i+1, e.name()), rList.Dx()-8-rw), rList.Min.X+3, y, c, 1)
		label(screen, right, rList.Max.X-3-rw, y, c, 1)
	}

	// bottom bar: ADD/REM, then an error or the position
	button(screen, rAdd, "ADD", g.down == rAdd)
	button(screen, rRem, "REM", g.down == rRem)
	info, c := fmt.Sprintf("%d/%d", min(g.cursor+1, len(s.List)), len(s.List)), cGreen
	if g.flash != "" {
		info, c = g.flash, cErr
	}
	if g.search.on {
		info, c = fmt.Sprintf("/%s_  %d", g.search.q, len(idx)), cWhite
	}
	box := image.Rect(rRem.Max.X+6, rAdd.Min.Y, gw-8, rAdd.Max.Y)
	fill(screen, box, cBlack)
	label(screen, fit(info, box.Dx()-6), box.Min.X+3, box.Min.Y+2, c, 1)
	if g.menu {
		button(screen, rAddFile, "FILE", g.down == rAddFile)
		button(screen, rAddDir, "DIR", g.down == rAddDir)
	}
	frame(screen, image.Rect(0, 0, gw, 116))
	frame(screen, image.Rect(0, 116, gw, gh))
	if g.help {
		fill(screen, image.Rect(0, rTitle.Max.Y, gw, gh), cBlack)
		for i, l := range helpLines(g.keys) {
			y := rTitle.Max.Y + 4 + i*rowH
			label(screen, l[0], 6, y, cGreen, 0.75)
			label(screen, l[1], gw-6-int(text.Advance(l[1], face)), y, cWhite, 1)
		}
		centered(screen, "press any key", image.Rect(0, gh-20, gw, gh-4), cDim)
	}
}
