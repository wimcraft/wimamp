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
