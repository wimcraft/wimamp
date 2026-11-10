package main

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

// playerAction returns the player call behind a key action, or nil when the action only affects the UI.
func playerAction(a string, cursor int) func() error {
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
