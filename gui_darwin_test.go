package main

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestDarwinPanelHelpers(t *testing.T) {
	nsApp() // loads AppKit, as the window does
	if s := goString(nsString("Ümlaut / пісня.m3u")); s != "Ümlaut / пісня.m3u" {
		t.Errorf("NSString round trip: %q", s)
	}
	if p := urlPath(fileURL("/tmp/a b/c.m3u")); p != "/tmp/a b/c.m3u" {
		t.Errorf("NSURL round trip: %q", p)
	}
	for _, c := range []string{"NSOpenPanel", "NSSavePanel", "UTType"} {
		if objc.GetClass(c) == 0 {
			t.Errorf("class %s missing", c)
		}
	}
	if objc.ID(objc.GetClass("UTType")).Send(sel("typeWithFilenameExtension:"), nsString("m3u")) == 0 {
		t.Error("no UTType for m3u")
	}
}
