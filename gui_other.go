//go:build !darwin

package main

import (
	"path/filepath"
	"strings"

	"github.com/ncruces/zenity"
)

func accessory() {}
func activate()  {}

// zenity's dialogs block their goroutine: run them off the UI thread.
const dialogsInProcess = false

func dialogOptions(title, dir string, exts []string) []zenity.Option {
	opts := []zenity.Option{zenity.Title(title)}
	if dir != "" {
		opts = append(opts, zenity.Filename(dir+string(filepath.Separator)))
	}
	if len(exts) > 0 {
		var pats []string
		for _, e := range exts {
			pats = append(pats, "*."+e)
		}
		opts = append(opts, zenity.FileFilter{Name: strings.Join(pats, " "), Patterns: pats, CaseFold: true})
	}
	return opts
}

func openDialog(title, dir string, folder, multi bool, exts []string) ([]string, error) {
	opts := dialogOptions(title, dir, exts)
	switch {
	case folder:
		p, err := zenity.SelectFile(append(opts, zenity.Directory())...)
		return []string{p}, err
	case multi:
		return zenity.SelectFileMultiple(opts...)
	}
	p, err := zenity.SelectFile(opts...)
	return []string{p}, err
}

func saveDialog(title, dir string, exts []string) (string, error) {
	return zenity.SelectFileSave(append(dialogOptions(title, dir, exts), zenity.ConfirmOverwrite())...)
}
