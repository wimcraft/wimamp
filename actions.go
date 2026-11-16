package main

import (
	"fmt"
	"slices"
)

// moveCursor applies a navigation action to a playlist cursor; ok is false for any other action.
func moveCursor(a string, cursor, page int, list []entry) (int, bool) {
	switch a {
	case "down":
		cursor++
	case "up":
		cursor--
	case "top":
		cursor = 0
	case "bottom":
		cursor = len(list) - 1
	case "halfdown":
		cursor += page / 2
	case "halfup":
		cursor -= page / 2
	case "current":
		cursor = currentIndex(list, cursor)
	default:
		return cursor, false
	}
	return max(min(cursor, len(list)-1), 0), true
}

// currentIndex is the playing entry, or def when nothing is playing.
func currentIndex(list []entry, def int) int {
	for i, e := range list {
		if e.Current {
			return i
		}
	}
	return def
}

// needsTrack lists the actions mpv rejects with an error while the playlist is empty.
var needsTrack = map[string]bool{
	"play": true, "resume": true, "toggle": true, "remove": true, "next": true, "prev": true,
	"seek-forward": true, "seek-back": true, "queue": true, "unqueue": true, "queue-later": true, "queue-earlier": true,
}

// playerAction returns the player call behind a key action, or nil when the action only affects
// the UI or needs a track and the playlist (n entries) is empty.
func playerAction(a string, cursor, n int) func() error {
	if n == 0 && needsTrack[a] {
		return nil
	}
	switch a {
	case "play":
		return func() error { return playIndex(cursor) }
	case "resume":
		return resume
	case "toggle":
		return toggle
	case "stop":
		return stop
	case "shuffle":
		return toggleShuffle
	case "repeat":
		return toggleRepeat
	case "next":
		return next
	case "queue":
		return func() error { return queueEdit(cursor, toggleID) }
	case "unqueue":
		return func() error { return queueEdit(cursor, unqueueID) }
	case "queue-later", "queue-earlier":
		d := 1
		if a == "queue-earlier" {
			d = -1
		}
		return func() error { return queueEdit(cursor, func(q []int, id int) []int { return moveID(q, id, d) }) }
	}
	c, ok := simple[a]
	if !ok || a == "quit" {
		c = map[string][]any{
			"remove":       {"playlist-remove", cursor},
			"seek-forward": {"seek", 5},
			"seek-back":    {"seek", -5},
			"volume-up":    {"add", "volume", 5},
			"volume-down":  {"add", "volume", -5},
		}[a]
	}
	if c == nil {
		return nil
	}
	return func() error { _, err := call(c); return err }
}

// queueMark is "[n] " for the n-th queued entry, shown before its name.
func queueMark(e entry, q []int) string {
	if i := slices.Index(q, e.ID); i >= 0 {
		return fmt.Sprintf("[%d] ", i+1)
	}
	return ""
}

// queueAction turns a key action in the queue view into the one to run: removing unqueues
// instead of deleting from the playlist, and queueing an entry that is already queued unqueues it.
func queueAction(a string) string {
	if a == "remove" || a == "queue" {
		return "unqueue"
	}
	return a
}

// onRow lists the actions that work on the selected row, so they do nothing without one.
var onRow = map[string]bool{"play": true, "remove": true, "queue": true, "unqueue": true, "queue-later": true, "queue-earlier": true}

// rowEntries is the entries at idx, the rows a list view shows.
func rowEntries(list []entry, idx []int) []entry {
	out := make([]entry, len(idx))
	for i, j := range idx {
		out[i] = list[j]
	}
	return out
}
