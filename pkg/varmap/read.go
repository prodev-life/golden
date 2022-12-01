package varmap

import (
	"golden/pkg/ryaml"
)

func Read(fileBaseNameOrDir string) VarMap {
	filenamesList := []string{}
	maps := ryaml.ReadYamlRecursive(fileBaseNameOrDir, func(filename string) interface{} {
		filenamesList = append(filenamesList, filename)
		m := New()
		return &m
	})
	merged := New()
	for i, m := range maps {
		vm := *m.(*VarMap)
		vm.SetSource(filenamesList[i])
		merged = Merge(merged, vm, ConflictResolutionError)
	}
	merged.SetPaths()
	return merged
}
