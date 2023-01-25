package goof

import (
	"os"
	"syscall"
)

// Restart the current application
// Attemtps to read the command line parameters from the current process
func Restart() {
	argv := os.Args
	if len(argv) > 0 {
		argv = argv[1:]
		os.StartProcess(argv[0], argv, &os.ProcAttr{
			Dir:   ".",
			Env:   os.Environ(),
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			Sys:   &syscall.SysProcAttr{},
		})
	}
}
