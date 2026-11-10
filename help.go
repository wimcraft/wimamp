package main

import (
	"sort"
	"strings"
)

// helpRows describes actions for `?` and `wimamp keys`; actions in one row are shown as "a / b".
var helpRows = []struct {
	desc    string
	actions []string
}{
	{"down / up", []string{"down", "up"}},
	{"top / bottom", []string{"top", "bottom"}},
	{"half page dn/up", []string{"halfdown", "halfup"}},
	{"jump to playing", []string{"current"}},
	{"search, jump to", []string{"search"}},
	{"play selected", []string{"play"}},
	{"play / resume", []string{"resume"}},
	{"pause on/off", []string{"toggle"}},
	{"stop", []string{"stop"}},
	{"next / prev track", []string{"next", "prev"}},
	{"seek +5s / -5s", []string{"seek-forward", "seek-back"}},
	{"volume up / down", []string{"volume-up", "volume-down"}},
	{"remove selected", []string{"remove"}},
	{"shuffle / repeat", []string{"shuffle", "repeat"}},
	{"add files / folder", []string{"add-files", "add-folder"}},
	{"this help", []string{"help"}},
	{"close", []string{"quit"}},
	{"quit player", []string{"quit-player"}},
}

// helpLines pairs each description with the keys currently bound to it.
func helpLines(keys map[string]string) [][2]string {
	by := map[string][]string{}
	for k, a := range keys {
		by[a] = append(by[a], k)
	}
	var out [][2]string
	for _, r := range helpRows {
		var parts []string
		for _, a := range r.actions {
			ks := by[a]
			sort.Slice(ks, func(i, j int) bool { // short keys first: "j down", not "down j"
				return len(ks[i]) < len(ks[j]) || len(ks[i]) == len(ks[j]) && ks[i] < ks[j]
			})
			if len(ks) == 0 {
				ks = []string{"-"}
			}
			parts = append(parts, strings.Join(ks, " "))
		}
		out = append(out, [2]string{r.desc, strings.Join(parts, " / ")})
	}
	return out
}
