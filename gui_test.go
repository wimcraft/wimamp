package main

import (
	"strings"
	"testing"
)

// The LCD clock is drawn from rectangles, because the bitmap font's ':' is two tiny '+'.
func TestLCDRects(t *testing.T) {
	s := "8:1"
	grid := make([][]byte, lcdH)
	for y := range grid {
		grid[y] = []byte(strings.Repeat(".", lcdWidth(s)))
	}
	for _, r := range lcdRects(s, 0, 0) {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				grid[y][x] = '#'
			}
		}
	}
	var b strings.Builder
	for _, row := range grid {
		b.WriteString(strings.TrimRight(string(row), ".") + "\n")
	}
	const want = `..######
..######
##......##.................##
##......##.................##
##......##....##...........##
##......##....##...........##
##......##.................##
##......##.................##
..######
..######
##......##.................##
##......##.................##
##......##....##...........##
##......##....##...........##
##......##.................##
##......##.................##
..######
..######
`
	if got := b.String(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestVolColor(t *testing.T) {
	for v, want := range map[float64]int{0: 0, 50: 2, 100: 4, 150: 4, -5: 0} {
		if got := volColor(v); got != volStops[want] {
			t.Errorf("volColor(%v) = %v, want %v", v, got, volStops[want])
		}
	}
}

func TestPlaylistScrollKeepsSelectionVisible(t *testing.T) {
	g := gui{s: state{List: make([]entry, 20)}, cursor: 13}
	g.keepVisible()
	_, sel, top := g.view()
	if sel != 13 || top != 1 {
		t.Fatalf("selection %d, top %d; want 13, 1", sel, top)
	}
	g.cursor = 0
	g.keepVisible()
	_, sel, top = g.view()
	if sel != 0 || top != 0 {
		t.Fatalf("selection %d, top %d; want 0, 0", sel, top)
	}
}

func TestScrollbarPreservesSelection(t *testing.T) {
	g := gui{s: state{List: make([]entry, 40)}, cursor: 2}
	bottom := rScroll.Max.Y - scrollThumbHeight
	g.scrollTo(bottom)
	g.clamp()
	_, sel, top := g.view()
	if sel != 2 || top != 27 || scrollThumb(40, top).Max.Y != rScroll.Max.Y {
		t.Fatalf("selection %d, top %d, thumb %v; want 2, 27, bottom", sel, top, scrollThumb(40, top))
	}
	g.scrollTo(rScroll.Min.Y)
	if g.top != 0 {
		t.Fatalf("scrolling to top gave %d", g.top)
	}
	g.search.on = true
	g.search.sel = 3
	g.scrollTo(bottom)
	g.clamp()
	_, sel, top = g.view()
	if sel != 3 || top != 27 {
		t.Fatalf("search selection %d, top %d; want 3, 27", sel, top)
	}
	g.s.List = nil
	g.scrollTo(bottom)
	g.clamp()
	if g.searchTop != 0 {
		t.Fatalf("empty search top = %d, want 0", g.searchTop)
	}
}

func TestMarqueeFitsFontHeight(t *testing.T) {
	m := face.Metrics()
	if height := m.HAscent + m.HDescent; height > float64(marqueeArea().Dy()) {
		t.Fatalf("marquee text height %.2f exceeds display height %d", height, marqueeArea().Dy())
	}
}

func TestPlayerActionEmptyPlaylist(t *testing.T) {
	for _, a := range []string{"play", "resume", "toggle", "remove", "next", "prev", "seek-forward"} {
		if playerAction(a, 0, 0) != nil {
			t.Errorf("%s runs on an empty playlist", a)
		}
		if playerAction(a, 0, 1) == nil {
			t.Errorf("%s does nothing with a track", a)
		}
	}
	for _, a := range []string{"stop", "shuffle", "repeat", "volume-up"} {
		if playerAction(a, 0, 0) == nil {
			t.Errorf("%s should still work on an empty playlist", a)
		}
	}
}
