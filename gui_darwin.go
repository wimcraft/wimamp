package main

import "github.com/ebitengine/purego/objc"

func nsApp() objc.ID {
	return objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
}

// accessory hides the Dock icon and Cmd-Tab entry, like Spotlight; the window can still take keyboard focus.
func accessory() { nsApp().Send(objc.RegisterName("setActivationPolicy:"), 1) }

// activate brings the window to the front with focus, also when it was started from a launcher.
func activate() { nsApp().Send(objc.RegisterName("activateIgnoringOtherApps:"), true) }
