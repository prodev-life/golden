package manifest

import (
	"golden/pkg/rerrors"
	"golden/pkg/ryaml"
)



type Manifest []string

type ManifestsCollection map[string]*Manifest

func NewManifestsCollection() ManifestsCollection {
	return ManifestsCollection{}
}

func ReadManifestsCollection(fileBaseNameOrDir string) ManifestsCollection {
	filenamesList := []string{}
	maps := ryaml.ReadYamlRecursive(fileBaseNameOrDir, func(filename string) interface{} {
		filenamesList = append(filenamesList, filename)
		return NewManifestsCollection()
	})

	c := NewManifestsCollection()
	sources := map[string]string{}
	for i, m := range maps {
		for name, manif := range m.(ManifestsCollection) {
			if _, ok := c[name]; ok {
				panic(rerrors.NewErrDuplicate(name, "instance", sources[name], filenamesList[i]))
			}
			c[name] = manif
			sources[name] = filenamesList[i]
		}
	}
	return c
}