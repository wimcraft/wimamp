package main

import (
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

	"github.com/hajimehoshi/bitmapfont/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/ncruces/zenity"
)

// Size in points: 1 logical pixel per point, which Retina still draws crisply at 2x.
const (
	gw, gh = 360, 408
	scale  = 1
	rowH   = 16 // bitmapfont line height
	rows   = 16
)

var (
	face   = text.NewGoXFace(bitmapfont.Face)
	cPanel = color.RGBA{0x2a, 0x2b, 0x3d, 0xff}
	cBar   = color.RGBA{0x18, 0x18, 0x28, 0xff}
	cLight = color.RGBA{0x5c, 0x5e, 0x7a, 0xff}
	cDark  = color.RGBA{0x10, 0x10, 0x18, 0xff}
	cBlack = color.RGBA{0x00, 0x00, 0x00, 0xff}
	cGreen = color.RGBA{0x00, 0xe8, 0x00, 0xff}
	cDim   = color.RGBA{0x00, 0x60, 0x00, 0xff}
	cWhite = color.RGBA{0xff, 0xff, 0xff, 0xff}
	cSel   = color.RGBA{0x10, 0x20, 0xb0, 0xff}
	cGold  = color.RGBA{0xd8, 0xc8, 0x60, 0xff}
	cErr   = color.RGBA{0xff, 0x60, 0x40, 0xff}
)

var (
	rTitle   = image.Rect(0, 0, gw, 14)
	rClose   = image.Rect(gw-13, 2, gw-3, 12)
	rLCD     = image.Rect(8, 18, 100, 60)
	rMarquee = image.Rect(106, 18, gw-8, 36)
	rVol     = image.Rect(106, 44, gw-87, 52)
	rShuf    = image.Rect(gw-79, 40, gw-45, 56)
	rRep     = image.Rect(gw-41, 40, gw-8, 56)
	rSeek    = image.Rect(8, 66, gw-8, 74)
	rPLTitle = image.Rect(0, 106, gw, 120)
	rList    = image.Rect(4, 122, gw-4, 122+rows*rowH)
	rAdd     = image.Rect(8, gh-24, 48, gh-6)
	rRem     = image.Rect(52, gh-24, 92, gh-6)
	rAddFile = image.Rect(8, gh-60, 48, gh-42)
	rAddDir  = image.Rect(8, gh-42, 48, gh-24)
)

// transport buttons, left to right, and the action each one runs
var transportActions = []string{"prev", "resume", "toggle", "stop", "next"}

func transport(i int) image.Rectangle { return image.Rect(8+i*26, 80, 31+i*26, 98) }

type gui struct {
	mu    sync.Mutex
	s     state
	err   error
	flash string
	dirty bool

	keys        map[string]string
	pending     string
	cursor, top int
	synced      bool

	drag         string // "win", "seek", "vol" while the left button is held
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
	case g.wasFocused && !f:
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
				g.do(playerAction("play", i))
			}
		case k == "esc":
			return g.closeWin()
		default:
			g.act(resolve(g.keys, &g.pending, k))
		}
	}
	g.mouse()
	g.clamp()
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
		if f := playerAction(a, g.cursor); f != nil {
			g.do(f)
		}
	}
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
		g.dirty = true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.dirty = true
		menu := g.menu
		g.menu = false
		switch {
		case menu && p.In(rAddFile):
			g.pick(false)
		case menu && p.In(rAddDir):
			g.pick(true)
		case p.In(rClose):
			g.quit = true
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
	case "seek":
		g.seekFrac = frac(x, rSeek)
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
		return idx, g.search.sel, max(0, g.search.sel-rows+1)
	}
	idx = make([]int, len(g.s.List))
	for i := range idx {
		idx[i] = i
	}
	return idx, g.cursor, g.top
}

func (g *gui) clamp() {
	g.cursor = max(min(g.cursor, len(g.s.List)-1), 0)
	g.top = max(min(g.top, g.cursor), g.cursor-rows+1, 0)
}

// --- drawing ---

func fill(dst *ebiten.Image, r image.Rectangle, c color.Color) {
	vector.FillRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
}

func bevel(dst *ebiten.Image, r image.Rectangle, sunken bool) {
	hi, lo := cLight, cDark
	if sunken {
		hi, lo = lo, hi
	}
	fill(dst, r, cPanel)
	fill(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1), hi)
	fill(dst, image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y), hi)
	fill(dst, image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y), lo)
	fill(dst, image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y), lo)
}

func label(dst *ebiten.Image, s string, x, y int, c color.Color, size float64) {
	op := &text.DrawOptions{}
	op.GeoM.Scale(size, size)
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
}

func centered(dst *ebiten.Image, s string, r image.Rectangle, c color.Color) {
	label(dst, s, r.Min.X+(r.Dx()-int(text.Advance(s, face)))/2, r.Min.Y+(r.Dy()-rowH)/2, c, 1)
}

func button(dst *ebiten.Image, r image.Rectangle, s string) {
	bevel(dst, r, false)
	centered(dst, s, r, cWhite)
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

func titleBar(dst *ebiten.Image, r image.Rectangle, s string) {
	fill(dst, r, cBar)
	w := int(text.Advance(s, face))
	mid := r.Min.Y + r.Dy()/2
	fill(dst, image.Rect(r.Min.X+4, mid-1, (r.Dx()-w)/2-4, mid+1), cGold)
	fill(dst, image.Rect((r.Dx()+w)/2+4, mid-1, r.Max.X-16, mid+1), cGold)
	centered(dst, s, r, cGold)
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

	titleBar(screen, rTitle, "WIMAMP")
	button(screen, rClose, "x")

	// LCD: state icon and elapsed time
	fill(screen, rLCD, cBlack)
	st, t := "playing", fmtTime(s.Pos)
	switch {
	case g.err != nil || s.Idle:
		st, t = "stopped", "-:--"
	case s.Paused:
		st = "paused"
	}
	icon(screen, st, image.Rect(rLCD.Min.X+2, rLCD.Min.Y+4, rLCD.Min.X+14, rLCD.Min.Y+16), cGreen)
	label(screen, t, rLCD.Max.X-4-int(text.Advance(t, face))*2, rLCD.Min.Y+6, cGreen, 2)

	// marquee: "N. Title (m:ss)", scrolling when it doesn't fit
	fill(screen, rMarquee, cBlack)
	msg, c := "stopped", cGreen
	if i := currentIndex(s.List, -1); g.err != nil {
		msg, c = "player not running", cErr
	} else if !s.Idle && i >= 0 {
		msg = fmt.Sprintf("%d. %s (%s)", i+1, s.Title, fmtTime(s.Dur))
	}
	area := rMarquee.Inset(3)
	g.scrolls = int(text.Advance(msg, face)) > area.Dx()
	if g.scrolls {
		rs := []rune(msg + "  ***  ")
		off := int(g.step % int64(len(rs)))
		msg = string(rs[off:]) + string(rs[:off])
	}
	label(screen.SubImage(area).(*ebiten.Image), msg, area.Min.X, area.Min.Y, c, 1)

	// volume, shuffle, repeat
	fill(screen, rVol.Inset(2), cDark)
	fill(screen, image.Rect(rVol.Min.X+2, rVol.Min.Y+2, rVol.Min.X+2+int(min(s.Vol, 100)/100*float64(rVol.Dx()-4)), rVol.Max.Y-2), cDim)
	kx := rVol.Min.X + int(min(s.Vol, 100)/100*float64(rVol.Dx()-6))
	bevel(screen, image.Rect(kx, rVol.Min.Y-2, kx+6, rVol.Max.Y+2), false)
	for _, t := range []struct {
		r  image.Rectangle
		s  string
		on bool
	}{{rShuf, "SHF", s.Shuffle}, {rRep, "REP", s.Repeat}} {
		bevel(screen, t.r, t.on)
		led := cDim
		if t.on {
			led = cGreen
		}
		fill(screen, image.Rect(t.r.Min.X+3, t.r.Min.Y+6, t.r.Min.X+6, t.r.Min.Y+9), led)
		label(screen, t.s, t.r.Min.X+9, t.r.Min.Y, cWhite, 1)
	}

	// seek bar
	fill(screen, rSeek, cDark)
	f := 0.0
	if g.drag == "seek" {
		f = g.seekFrac
	} else if s.Dur > 0 {
		f = s.Pos / s.Dur
	}
	sx := rSeek.Min.X + int(f*float64(rSeek.Dx()-12))
	bevel(screen, image.Rect(sx, rSeek.Min.Y-1, sx+12, rSeek.Max.Y+1), false)

	for i, a := range transportActions {
		bevel(screen, transport(i), false)
		icon(screen, a, transport(i), cWhite)
	}

	// playlist
	titleBar(screen, rPLTitle, "PLAYLIST")
	fill(screen, rList, cBlack)
	if len(s.List) == 0 {
		label(screen, "empty - a: add files, ?: keys", rList.Min.X+4, rList.Min.Y, cDim, 1)
	}
	idx, sel, top := g.view()
	for n := top; n < len(idx) && n < top+rows; n++ {
		i := idx[n]
		e := s.List[i]
		y := rList.Min.Y + (n-top)*rowH
		if n == sel {
			fill(screen, image.Rect(rList.Min.X, y, rList.Max.X, y+rowH), cSel)
		}
		c, right := cGreen, ""
		if e.Current {
			c = cWhite
			if s.Dur > 0 {
				right = fmtTime(s.Dur)
			}
		}
		rw := int(text.Advance(right, face))
		label(screen, fit(fmt.Sprintf("%d. %s", i+1, e.name()), rList.Dx()-8-rw), rList.Min.X+3, y, c, 1)
		label(screen, right, rList.Max.X-3-rw, y, c, 1)
	}

	// bottom bar: ADD/REM, then an error or the position
	button(screen, rAdd, "ADD")
	button(screen, rRem, "REM")
	info, c := fmt.Sprintf("%d/%d", g.cursor+1, len(s.List)), cGreen
	if g.flash != "" {
		info, c = g.flash, cErr
	}
	if g.search.on {
		info, c = fmt.Sprintf("/%s_  %d", g.search.q, len(idx)), cWhite
	}
	box := image.Rect(rRem.Max.X+6, rAdd.Min.Y, gw-8, rAdd.Max.Y)
	fill(screen, box, cBlack)
	label(screen, fit(info, box.Dx()-6), box.Min.X+3, box.Min.Y+1, c, 1)
	if g.menu {
		button(screen, rAddFile, "FILE")
		button(screen, rAddDir, "DIR")
	}
	if g.help {
		fill(screen, image.Rect(0, rTitle.Max.Y, gw, gh), cBlack)
		for i, l := range helpLines(g.keys) {
			y := rTitle.Max.Y + 4 + i*rowH
			label(screen, l[0], 6, y, cGreen, 1)
			label(screen, l[1], gw-6-int(text.Advance(l[1], face)), y, cWhite, 1)
		}
		centered(screen, "press any key", image.Rect(0, gh-20, gw, gh-4), cDim)
	}
}
