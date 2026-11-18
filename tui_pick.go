package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type pickedMsg []string // paths chosen in fzf; loaded in Update, since this callback runs on the UI loop

// pick hands the terminal to fzf to choose files or folders under the home folder, then appends them.
func pick(dir bool) tea.Cmd {
	if _, err := exec.LookPath("fzf"); err != nil {
		return func() tea.Msg { return flashMsg("install fzf to add music here, or run: wimamp add <path>") }
	}
	c, out := fzfCmd(dir)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		var ee *exec.ExitError
		if errors.As(err, &ee) && (ee.ExitCode() == 1 || ee.ExitCode() == 130) { // no match, or esc
			return nil
		}
		if err != nil {
			return flashMsg("fzf: " + err.Error())
		}
		return pickedMsg(strings.Split(strings.TrimRight(out.String(), "\r\n"), "\n"))
	})
}

func fzfCmd(dir bool) (*exec.Cmd, *bytes.Buffer) {
	walker, prompt := "file,follow", "add files> "
	if dir {
		walker, prompt = "dir,follow", "add folder> "
	}
	home, _ := os.UserHomeDir()
	c := exec.Command("fzf", "--multi", "--walker="+walker, "--walker-root="+home, "--walker-skip=.git,node_modules,Library,AppData", "--prompt="+prompt,
		"--header=tab selects several, enter adds, esc cancels")
	c.Env = append(os.Environ(), "FZF_DEFAULT_COMMAND=") // a user's default command would replace the walker
	out := &bytes.Buffer{}
	c.Stdout = out
	return c, out
}
