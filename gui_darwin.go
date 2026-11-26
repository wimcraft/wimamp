package main

import (
	"github.com/ebitengine/purego/objc"
	"github.com/ncruces/zenity"
)

func nsApp() objc.ID {
	return objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
}

// accessory hides the Dock icon and Cmd-Tab entry, like Spotlight; the window can still take keyboard focus.
func accessory() { nsApp().Send(objc.RegisterName("setActivationPolicy:"), 1) }

// activate brings the window to the front with focus, also when it was started from a launcher.
// Activating the app is not enough after one of our own panels closed: the app is already active
// but no window is key, so the window is made key too.
func activate() {
	app := nsApp()
	app.Send(sel("activateIgnoringOtherApps:"), true)
	ws := app.Send(sel("windows"))
	panel := objc.ID(objc.GetClass("NSPanel"))
	for i := range objc.Send[uint](ws, sel("count")) {
		w := ws.Send(sel("objectAtIndex:"), i)
		if objc.Send[bool](w, sel("isVisible")) && !objc.Send[bool](w, sel("isKindOfClass:"), panel) {
			w.Send(sel("makeKeyAndOrderFront:"), 0)
			return
		}
	}
}

// File dialogs run in this process as NSOpenPanel / NSSavePanel. zenity runs them in a separate osascript
// process, which doesn't always get keyboard focus and hands focus to another app when it closes,
// so the window lost focus and closed. They are modal and must run on the main thread, where Update runs.

const dialogsInProcess = true

func sel(s string) objc.SEL { return objc.RegisterName(s) }

func nsString(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(sel("stringWithUTF8String:"), s)
}

func goString(s objc.ID) string { return objc.Send[string](s, sel("UTF8String")) }

func fileURL(p string) objc.ID {
	return objc.ID(objc.GetClass("NSURL")).Send(sel("fileURLWithPath:"), nsString(p))
}

func urlPath(u objc.ID) string { return goString(u.Send(sel("path"))) }

// setup applies what open and save panels share; exts limits the files that can be chosen.
func setupPanel(p objc.ID, title, dir string, exts []string) {
	p.Send(sel("setMessage:"), nsString(title))
	if dir != "" {
		p.Send(sel("setDirectoryURL:"), fileURL(dir))
	}
	if len(exts) == 0 {
		return
	}
	types := objc.ID(objc.GetClass("NSMutableArray")).Send(sel("array"))
	ut := objc.ID(objc.GetClass("UTType")) // macOS 11+; Go itself needs macOS 12
	for _, e := range exts {
		if t := ut.Send(sel("typeWithFilenameExtension:"), nsString(e)); t != 0 {
			types.Send(sel("addObject:"), t)
		}
	}
	p.Send(sel("setAllowedContentTypes:"), types)
}

const modalOK = 1

func openDialog(title, dir string, folder, multi bool, exts []string) ([]string, error) {
	p := objc.ID(objc.GetClass("NSOpenPanel")).Send(sel("openPanel"))
	p.Send(sel("setCanChooseFiles:"), !folder)
	p.Send(sel("setCanChooseDirectories:"), folder)
	p.Send(sel("setAllowsMultipleSelection:"), multi)
	setupPanel(p, title, dir, exts)
	if objc.Send[int](p, sel("runModal")) != modalOK {
		return nil, zenity.ErrCanceled
	}
	urls := p.Send(sel("URLs"))
	var out []string
	for i := range objc.Send[uint](urls, sel("count")) {
		out = append(out, urlPath(urls.Send(sel("objectAtIndex:"), i)))
	}
	return out, nil
}

// saveDialog asks for a file to write; the panel itself asks before overwriting and adds the extension.
func saveDialog(title, dir string, exts []string) (string, error) {
	p := objc.ID(objc.GetClass("NSSavePanel")).Send(sel("savePanel"))
	setupPanel(p, title, dir, exts)
	if objc.Send[int](p, sel("runModal")) != modalOK {
		return "", zenity.ErrCanceled
	}
	return urlPath(p.Send(sel("URL"))), nil
}
