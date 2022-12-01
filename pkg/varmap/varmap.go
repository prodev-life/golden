package varmap

import (
	"fmt"
	"golden/pkg/rerrors"
	"golden/pkg/rtemplate"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type VarMap map[string]*Var

type Var struct {
	Value  interface{}
	Path   *Path
	Source string
}

func New() VarMap {
	m := make(VarMap)
	return m
}

type ConflictResolution int

const (
	ConflictResolutionOverride ConflictResolution = iota
	ConflictResolutionError
)

func merge(commonPath *Path, lower, higher VarMap, cr ConflictResolution) VarMap {
	merged := lower
	for higherK, higherV := range higher {
		if lowerV, ok := lower[higherK]; ok {
			lowerSubMap, isVarMap := lowerV.Value.(VarMap)

			if isVarMap {
				higherSubMap, alsoVarMap := higherV.Value.(VarMap)
				if !alsoVarMap {
					panic(rerrors.NewErrStringf(
						"Variables types mismatch:\n%s: [%s] - a map\n%s [%s] - not a map",
						lowerV.Path.String(), lowerV.Source,
						higherV.Path.String(), higherV.Source))
				}
				thisPath := commonPath.CopyJoin(higherK)
				subMerged := merge(commonPath.Join(higherK), lowerSubMap, higherSubMap, cr)
				merged[higherK] = &Var{subMerged, thisPath, higherV.Source}
				continue
			}

			switch cr {
			case ConflictResolutionOverride:
				merged[higherK] = higherV
				continue
			case ConflictResolutionError:
				conflictPath := commonPath.Join(higherK)
				panic(&ResolutionError{
					Path:    *conflictPath,
					Sources: [2]string{lowerV.Source, higherV.Source},
				})
			}
			panic("unreachable")
		}
		merged[higherK] = higherV
		continue
	}
	return merged
}

func Merge(lower, higher VarMap, cr ConflictResolution) VarMap {
	return merge(NewPath(), lower, higher, cr)
}

func (v *Var) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.MappingNode {
		m := New()
		err := m.UnmarshalYAML(node)
		if err != nil {
			return err
		}
		v.Value = m
		return nil
	}
	var anything interface{}
	err := node.Decode(&anything)
	if err != nil {
		return err
	}
	if anything == nil {
		v.Value = ""
		return nil
	}
	v.Value = anything
	return nil
}

func (m *VarMap) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("expected a map")
	}
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		var value Var
		valueNode := node.Content[i+1]
		err := value.UnmarshalYAML(valueNode)
		if err != nil {
			return err
		}
		(*m)[key] = &value
	}
	return nil
}

func (m *VarMap) CustomUnmarshallYAML(data []byte) error {
	type Doc struct {
		Root VarMap `yaml:",inline"`
	}
	d := Doc{New()}
	err := yaml.Unmarshal(data, &d)
	if err != nil {
		return err
	}
	*m = d.Root
	for k, v := range *m {
		if v == nil {
			(*m)[k] = &Var{Value: ""}
		}
	}
	return nil
}

func (m VarMap) SetSource(filename string) {
	for _, v := range m {
		v.Source = filename
		if vm, ok := v.Value.(VarMap); ok {
			vm.SetSource(filename)
		}
	}
}

func (m VarMap) setPaths(commonPath *Path) {
	for k, v := range m {
		v.Path = commonPath.CopyJoin(k)
		if vm, ok := v.Value.(VarMap); ok {
			vm.setPaths(commonPath.CopyJoin(k))
		}
	}
}

func (m VarMap) SetPaths() {
	m.setPaths(NewPath())
}

func (m VarMap) toRegularMap() map[string]interface{} {
	reg := make(map[string]interface{})
	for k, v := range m {
		if vm, ok := v.Value.(VarMap); ok {
			regsubmap := vm.toRegularMap()
			reg[k] = regsubmap
			continue
		}
		reg[k] = v.Value
	}
	return reg
}

func setRegularMapValue(topMap map[string]interface{}, path *Path, val interface{}) {
	m := topMap
	for i, el := range path.Elements {
		if i != len(path.Elements)-1 {
			m = m[el].(map[string]interface{})
			continue
		}
		m[el] = val
	}
}

func substituteTemplatedVars(templatedVars map[*Var]struct{}, topMap map[string]interface{}) {
	for tv, _ := range templatedVars {
		v := tv.Value.(string)

		tmpl := template.Must(rtemplate.New("vartemplate").Parse(v))
		buf := strings.Builder{}
		err := tmpl.Execute(&buf, topMap)
		if err != nil {
			panic(rtemplate.NewErrExec(tv.Source, "resolving variable: "+tv.Path.String(), err))
		}
		newVal := buf.String()
		setRegularMapValue(topMap, tv.Path, newVal)
		newTmpl, err := rtemplate.New("vartemplate").Parse(newVal)
		if err != nil {
			panic(rtemplate.NewErrParse(fmt.Sprintf("%s: %s", tv.Source, tv.Path.String()), err))
		}
		if !rtemplate.IsTemplate(newTmpl) {
			delete(templatedVars, tv)
		}
	}
}

func (topMap VarMap) getAllTemplatedVarsWithTheirMaps() map[*Var]struct{} {
	out := make(map[*Var]struct{})
	queue := make([]VarMap, 0, 30)
	queue = append(queue, topMap)
	for len(queue) != 0 {
		m := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		for _, v := range m {
			if str, ok := v.Value.(string); ok {
				tmpl, err := rtemplate.New("vartemplate").Parse(str)
				if err != nil {
					panic(rtemplate.NewErrParse(fmt.Sprintf("%s: %s", v.Source, v.Path.String()), err))
				}
				if rtemplate.IsTemplate(tmpl) {
					out[v] = struct{}{}
				}
				continue
			}
			if subMap, ok := v.Value.(VarMap); ok {
				queue = append(queue, subMap)
				continue
			}
		}
	}
	return out
}

type ErrCyclicDependency struct {
	vars map[*Var]struct{}
}

func (e *ErrCyclicDependency) NiceError() string {
	buf := strings.Builder{}
	buf.WriteString("Cyclic dependency detected within templated variables:")
	for tv := range e.vars {
		absPath, err := filepath.Abs(tv.Source)
		if err != nil {
			panic(err)
		}
		buf.WriteString(fmt.Sprintf("\n\t%s: defined in file - %s", tv.Path.String(), absPath))
	}
	return buf.String()
}

func (m VarMap) SubstituteTemplatedVars() map[string]interface{} {
	regmap := m.toRegularMap()
	templatedVars := m.getAllTemplatedVarsWithTheirMaps()
	templatedVarsCount := len(templatedVars)
	for templatedVarsCount != 0 {
		substituteTemplatedVars(templatedVars, regmap)
		if len(templatedVars) == templatedVarsCount {
			panic(&ErrCyclicDependency{templatedVars})
		}
		templatedVarsCount = len(templatedVars)
	}

	return regmap
}
