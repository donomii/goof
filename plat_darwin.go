package goof

import (
	"os"
	"syscall"
)

//Restart the current application
//Attemtps to read the command line parameters from the current process
func Restart() {
	procAttr := new(syscall.ProcAttr)
	procAttr.Files = []uintptr{0, 1, 2}
	procAttr.Dir = os.Getenv("PWD")
	procAttr.Env = os.Environ()
	exe, _ := os.Executable()
	syscall.ForkExec(exe, os.Args, procAttr)
}
