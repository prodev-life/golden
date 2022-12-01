package sh

import (
	"fmt"
	"os"
	"os/exec"
)

type ErrCmd struct {
	cmd string
	exitCode int
	combinedOut []byte
}

func newErrCmd(cmd string, out []byte, err error) *ErrCmd {
	if eerr, ok := err.(*exec.ExitError); ok {
		return &ErrCmd{cmd, eerr.ExitCode(), out}
	}
	panic(err)
}

func (err *ErrCmd) NiceError() string {
	return fmt.Sprintf(
		"COMMAND: %s\nexited with status %d\nOUTPUT:\n%s",
		err.cmd, err.exitCode, err.combinedOut,
	)
}


func MustDo(command string) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}


func MustDof(format string, args ...interface{}) {
	MustDo(fmt.Sprintf(format, args...))
}


func MustDoSilently(command string) {
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(newErrCmd(command, out, err))
	}
}

func MustDoSilentlyf(format string, args ...interface{}) {
	MustDoSilently(fmt.Sprintf(format, args...))
}

func MustCp(src, dest string) {
	MustDoSilentlyf("cp %s %s", src, dest)
}

type shellT struct {}

func (s shellT) MustDoSilentlyf(format string, args ...interface{}) {
	MustDoSilentlyf(format, args...)
}

func (s shellT) MustCp(src, dest string) {
	MustCp(src, dest)
}

var Shell shellT