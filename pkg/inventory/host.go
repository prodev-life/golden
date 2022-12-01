package inventory

import (
	"fmt"
	"golden/pkg/rerrors"
	"golden/pkg/ryaml"
	"os/user"
)

type Host struct {
	SshConfigHost string `yaml:"ssh_config_host"`
	SshHostname   string `yaml:"ssh_hostname"`
	SshUser       string `yaml:"ssh_user"`
}

func (h *Host) IsLocalHost() bool {
	if h.SshConfigHost != "" {
		return false
	}
	if h.SshHostname == "localhost" || h.SshHostname == "127.0.0.1" || h.SshHostname == "" {
		return true
	}
	return false
}

func (h *Host) IsThisUser() bool {
	if !h.IsLocalHost() {
		return false
	}
	if h.SshUser == "" {
		return true
	}
	user, err := user.Current()
		if err != nil {
			panic(err)
		}

	return h.SshUser == user.Username
}

func (h *Host) GetUser() string {
	if !h.IsLocalHost() {
		panic("GetUser for remote host is not a valid operation")
	}
	if h.SshUser == "" {
		user, err := user.Current()
		if err != nil {
			panic(err)
		}

		username := user.Username
		return username
	}
	return h.SshUser
}

func (h *Host) GetSshConnStr() string {
	if h.IsLocalHost() {
		panic("GetSshConnStr for localhost is not a valid operation")
	}
	if h.SshConfigHost != "" {
		return h.SshConfigHost
	}
	return fmt.Sprintf("%s@%s", h.SshUser, h.SshHostname)
}

func (h *Host) String() string {
	if h.IsThisUser() {
		return "[local]"
	}
	if h.IsLocalHost() {
		return fmt.Sprintf("[sudo -iu %s]", h.SshUser)
	}
	return fmt.Sprintf("[ssh %s]", h.GetSshConnStr())
}

type HostsCollection map[string]*Host

func ReadHosts(fileBaseNameOrDir string) HostsCollection {
	filenamesList := []string{}
	maps := ryaml.ReadYamlRecursive(fileBaseNameOrDir, func(filename string) interface{} {
		filenamesList = append(filenamesList, filename)
		return HostsCollection{}
	})
	merged := HostsCollection{}
	sources := map[string]string{}
	for i, m := range maps {
		inv := m.(HostsCollection)
		for k, v := range inv {
			if _, ok := merged[k]; ok {
				panic(rerrors.NewErrDuplicate(k, "host definition", sources[k], filenamesList[i]))
			}
			merged[k] = v
			sources[k] = filenamesList[i]
		}
	}
	return merged
}
