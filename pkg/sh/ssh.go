package sh

import "fmt"

type SshSession struct {
	controlPath string
	connStr string
}

func NewSshSession(controlPath string, connStr string) *SshSession {
	ssh := &SshSession{controlPath: controlPath, connStr: connStr}
	MustDof("ssh -M -o \"ControlPath=%s\" -o \"ControlPersist=yes\" %s true", controlPath, connStr)
	return ssh
}

func (ssh *SshSession) Close() {
	MustDoSilentlyf("ssh -o \"ControlPath=%s\" -O exit %s", ssh.controlPath, ssh.connStr)
}

func (ssh *SshSession) MustCp(src, dest string) {
	MustDoSilentlyf("scp -o \"ControlPath=%s\" %s %s:%s", ssh.controlPath, src, ssh.connStr, dest)
}

func (ssh *SshSession) MustDoSilentlyf(format string, args... interface{}) {
	MustDoSilentlyf("ssh -S %s %s %s", ssh.controlPath, ssh.connStr, fmt.Sprintf(format, args...))
}