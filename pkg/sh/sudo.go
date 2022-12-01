package sh

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Sudo struct {
	user string
	userHomeDir string
}

func NewSudo(user string) Sudo {
	getHome := fmt.Sprintf("echo ~%s", user)
	cmd := exec.Command("sh", "-c", getHome)
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(newErrCmd(getHome, out, err))
	}
	homeDir := strings.TrimSpace(string(out))
	return Sudo{user: user, userHomeDir: homeDir}
}

func (s Sudo) MustDoSilentlyf(format string, args ...interface{}) {
	MustDoSilentlyf("sudo -iu %s %s", s.user, fmt.Sprintf(format, args...))
}

func (s Sudo) MustCp(src, dest string) {
	s.MustDoSilentlyf("cp %s %s", src, filepath.Join(s.userHomeDir, dest))
}