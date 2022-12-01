package inventory

import (
	"golden/pkg/manifest"
	"golden/pkg/rerrors"
	"path/filepath"
)

type Inventory struct {
	instances      InstancesCollection
	groups         GroupsCollection
	hosts          HostsCollection
	hostInstances  map[string][]*Instance
	instanceGroups map[string][]string
}

func (inv *Inventory) GetInstancesForManifest(names manifest.Manifest) []*Instance {
	outMap := map[string]struct{}{}
	out := make([]*Instance, 0, len(names))

	for _, name := range names {
		if inst, ok := inv.instances[name]; ok {
			if _, present := outMap[name]; present {
				continue
			}
			outMap[name] = struct{}{}
			out = append(out, inst)
			continue
		}
		if _, ok := inv.hosts[name]; ok {
			if insts, ok := inv.hostInstances[name]; ok {
				for _, inst := range insts {
					if _, ok := outMap[inst.Name]; !ok {
						outMap[inst.Name] = struct{}{}
						out = append(out, inst)
					}
				}
			}
		}
		if group, ok := inv.groups[name]; ok {
			for _, instName := range group.List() {
				if _, ok := outMap[instName]; !ok {
					outMap[instName] = struct{}{}
					out = append(out, inv.instances[instName])
				}
			}
		}
	}

	return out
}

func (inv *Inventory) GetHost(host string) *Host {
	return inv.hosts[host]
}

func (inv *Inventory) GetGroups(instance string) []string {
	return inv.instanceGroups[instance]
}

func (inv *Inventory) GetAllInstances() InstancesCollection {
	return inv.instances
}

func New() *Inventory {
	return &Inventory{
		instances:      map[string]*Instance{},
		groups:         map[string]*Group{},
		hosts:          map[string]*Host{},
		hostInstances:  map[string][]*Instance{},
		instanceGroups: map[string][]string{},
	}
}

func (inv *Inventory) SetHostsToLocalhost() {
	for _, h := range inv.hosts {
		*h = Host{}
	}
}

func (inv *Inventory) OverrideInstallPrefix(overrides map[string]string) {
	for k, prefix := range overrides {
		inv.instances[k].InstallPrefix = prefix
	}
}

func ReadInventory(rootDir string) *Inventory {
	inv := New()

	inv.instances = ReadInstancesCollection(filepath.Join(rootDir, "instances"))
	inv.hosts = ReadHosts(filepath.Join(rootDir, "hosts"))
	inv.groups = ReadGroupsCollection(filepath.Join(rootDir, "groups"))
	inv.MustHaveUniqueNames()

	// Forming inv.hostInstances
	for _, inst := range inv.instances {
		host := inst.Host
		if insts, ok := inv.hostInstances[host]; ok {
			inv.hostInstances[host] = append(insts, inst)
		} else {
			inv.hostInstances[host] = []*Instance{inst}
		}
	}

	instanceGroupsAsMap := map[string]map[string]struct{}{}
	hostGroups := map[string][]string{}

	// Forming hostGroups and direct instanceGroups
	for grName, group := range inv.groups {
		for name := range group.instances {

			if _, isHost := inv.hosts[name]; isHost {
				if groups, ok := hostGroups[name]; ok {
					hostGroups[name] = append(groups, grName)
				} else {
					hostGroups[name] = []string{grName}
				}
				continue
			}

			if _, isInstance := inv.instances[name]; isInstance {
				if _, ok := instanceGroupsAsMap[name]; ok {
					instanceGroupsAsMap[name][grName] = struct{}{}
				} else {
					instanceGroupsAsMap[name] = map[string]struct{}{grName: struct{}{}}
				}
				continue
			}
			panic(rerrors.NewErrStringf("%s is not a instance/host, but specified in group %s", name, grName))
		}
	}

	// Group inheritance for instances via hosts
	for host, insts := range inv.hostInstances {
		groups, ok := hostGroups[host]
		if !ok {
			continue
		}
		for _, inst := range insts {
			for _, grName := range groups {
				if _, ok := instanceGroupsAsMap[inst.Name]; ok {
					if _, isPresent := instanceGroupsAsMap[inst.Name][grName]; isPresent {
						continue
					}
					instanceGroupsAsMap[inst.Name][grName] = struct{}{}
				} else {
					instanceGroupsAsMap[inst.Name] = map[string]struct{}{grName: struct{}{}}
				}
			}
		}
	}

	// Now instanceGroups are full, we reinit groups to contain only instances

	for _, gr := range inv.groups {
		gr.instances = map[string]struct{}{}
		gr.ordered = []string{}
	}
	for inst, groups := range instanceGroupsAsMap {
		inv.instanceGroups[inst] = []string{}
		for grName := range groups {
			inv.instanceGroups[inst] = append(inv.instanceGroups[inst], grName)
			inv.groups[grName].instances[inst] = struct{}{}
			inv.groups[grName].ordered = append(inv.groups[grName].ordered, inst)
		}
	}

	return inv
}

func (inv *Inventory) MustHaveUniqueNames() {

	names := map[string]struct{}{}

	for n, _ := range inv.instances {
		if _, ok := names[n]; ok {
			panic(rerrors.NewErrDuplicate(n, "instance/host/group"))
		}
		names[n] = struct{}{}
	}
	for n, _ := range inv.hosts {
		if _, ok := names[n]; ok {
			panic(rerrors.NewErrDuplicate(n, "instance/host/group"))
		}
		names[n] = struct{}{}
	}
	for n, _ := range inv.groups {
		if _, ok := names[n]; ok {
			panic(rerrors.NewErrDuplicate(n, "instance/host/group"))
		}
		names[n] = struct{}{}
	}

}
