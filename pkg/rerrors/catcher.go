package rerrors

import (
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"text/template"
)

func Recover(recovered interface{}, ok *bool) {
	if recovered == nil {
		*ok = true
		return
	}

	*ok = false

	switch err := recovered.(type) {
	case NiceError:
		fmt.Fprintln(os.Stderr, err.NiceError())
		return
	case *os.PathError:
		fmt.Fprintln(os.Stderr, err.Error())
		return
	case *exec.ExitError:
		fmt.Fprintf(os.Stderr, "Subcommand exited with satus: %d\n", err.ExitCode())
		return
	case template.ExecError:
		fmt.Fprintf(os.Stderr, "Template execution failed:\n%s\n", err.Err.Error())
		return
	}


	fmt.Fprintf(
		os.Stderr,
		"Unexpected error occurred!\nError: %v\nFile a bug report: %s\nStack:%s",
		recovered, "github.com/prodev-live/golden/issues", debug.Stack(),
	)
}