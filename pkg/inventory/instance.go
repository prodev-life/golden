package inventory

import (
	"golden/pkg/rerrors"
	"golden/pkg/ryaml"
)

type Instance struct {
	Name          string `yaml:"name"`
	Host          string `yaml:"host"`
	App           string `yaml:"app"`
	InstallPrefix string `yaml:"install_prefix"`
}

type InstancesCollection map[string]*Instance

func NewInstancesCollection() InstancesCollection {
	return InstancesCollection{}
}

func ReadInstancesCollection(fileBaseNameOrDir string) InstancesCollection {
	filenamesList := []string{}
	maps := ryaml.ReadYamlRecursive(fileBaseNameOrDir, func(filename string) interface{} {
		filenamesList = append(filenamesList, filename)
		return NewInstancesCollection()
	})

	c := NewInstancesCollection()
	sources := map[string]string{}
	for i, m := range maps {
		for name, inst := range m.(InstancesCollection) {
			inst.Name = name
			if _, ok := c[name]; ok {
				panic(rerrors.NewErrDuplicate(inst.Name, "instance", sources[name], filenamesList[i]))
			}
			c[name] = inst
			sources[name] = filenamesList[i]
		}
	}
	return c
}
