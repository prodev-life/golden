package inventory

import (
	"golden/pkg/rerrors"
	"golden/pkg/ryaml"

	"gopkg.in/yaml.v3"
)

type Group struct {
	ordered []string
	instances map[string]struct{}
}

func (g *Group) List() []string {
	return g.ordered
}

func (g *Group) Has(inst string) bool {
	_, ok := g.instances[inst]
	return ok
}

func (gr *Group) UnmarshalYAML(node *yaml.Node) error {
	gr.instances = map[string]struct{}{}
	lst := []string{}
	node.Decode(&lst)
	gr.ordered = make([]string, 0, len(lst))
	for _, name := range lst {
		if _, ok := gr.instances[name]; ok {
			return rerrors.NewErrDuplicate(name, "instance or host in group")
		}
		gr.instances[name] = struct{}{}
		gr.ordered = append(gr.ordered, name)
	}
	return nil
}

type GroupsCollection map[string]*Group

func NewGroupsCollection () GroupsCollection {
	return GroupsCollection{}
}

func ReadGroupsCollection(fileBaseNameOrDir string) GroupsCollection {
	filenamesList := []string{}
	maps := ryaml.ReadYamlRecursive(fileBaseNameOrDir, func(filename string) interface{} {
		filenamesList = append(filenamesList, filename)
		return NewGroupsCollection()
	})

	c := NewGroupsCollection()
	sources := map[string]string{}
	for i, m := range maps {
		for name, gr := range m.(GroupsCollection) {
			if _, ok := c[name]; ok {
				panic(rerrors.NewErrDuplicate(name, "group", sources[name], filenamesList[i]))
			}
			c[name] = gr
			sources[name] = filenamesList[i]
		}
	}
	return c
}

type ErrRepeatingGroup struct {
	filename string
}