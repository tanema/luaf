package lfile

import (
	"bufio"
	"io"
	"os/exec"
	"runtime"
)

// PopenCommand creates a platform independent exec.Cmd.
func PopenCommand(arg string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("C:\\Windows\\system32\\cmd.exe", append([]string{"/c"}, arg)...)
	}
	return exec.Command("/bin/sh", append([]string{"-c"}, arg)...)
}

// POpen will create a new command and executes it with a filewrapper around it,
// which makes it easy to read and write from.
func POpen(cmdSrc, mode string) (*File, error) {
	cmd := PopenCommand(cmdSrc)
	newFile := &File{Path: cmdSrc}
	switch mode {
	case "r":
		stderr, _ := cmd.StderrPipe()
		stdout, _ := cmd.StdoutPipe()
		newFile.reader = bufio.NewReader(io.MultiReader(stdout, stderr))
		newFile.readOnly = true
	case "w":
		stdin, _ := cmd.StdinPipe()
		newFile.handle = ostoFile(stdin)
		newFile.writeOnly = true
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	newFile.process = cmd.Process
	return newFile, nil
}
