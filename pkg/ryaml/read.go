package ryaml

import (
	"golden/pkg/fsys"
	"golden/pkg/rerrors"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func ReadYamlFile(filename string, out interface{}) {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(rerrors.NewErrIo(filename, "reading yaml", err))
	}
	if unm, ok := out.(Unmarshaller); ok {
		err = unm.CustomUnmarshallYAML(data)
	} else {
		err = yaml.Unmarshal(data, out)
	}
	if err != nil {
		panic(rerrors.NewErrIo(filename, "parsing yaml", err))
	}
}

func ReadYamlRecursive(fileBaseNameOrDir string, placeholderGenerator func(filename string) interface{}) []interface{} {
	var filename string
	var dirname string
	if !strings.HasSuffix(fileBaseNameOrDir, ".yml") {
		filename = fileBaseNameOrDir + ".yml"
		dirname = fileBaseNameOrDir
	} else {
		filename = fileBaseNameOrDir
		dirname = filename[0 : len(filename)-len(".yml")]
	}

	dirExists := fsys.DoesDirExists(dirname)
	fileExists := fsys.DoesFileExists(filename)

	if dirExists && fileExists {
		panic(rerrors.NewErrStringf("ambiguous %s OR %s", dirname, filename))
	}

	if !dirExists && !fileExists {
		return nil
	}

	if fsys.DoesDirExists(dirname) {
		allFiles, err := fsys.GetAllFilesRecursive(dirname)
		if err != nil {
			panic(err)
		}
		out := make([]interface{}, 0, len(allFiles))
		for _, file := range allFiles {
			if strings.HasSuffix(file, ".yml") {
				ph := placeholderGenerator(file)
				ReadYamlFile(file, ph)
				out = append(out, ph)
			}
		}
		return out
	}
	ph := placeholderGenerator(filename)
	ReadYamlFile(filename, ph)
	return []interface{}{ph}
}

type Unmarshaller interface {
	CustomUnmarshallYAML(data []byte) error
}
