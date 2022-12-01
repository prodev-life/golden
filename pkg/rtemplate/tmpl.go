package rtemplate

import (
	"fmt"
	"strings"
	"text/template"
	"text/template/parse"

	"gopkg.in/yaml.v3"
)

func IsTemplate(t *template.Template) bool {
	if t.Tree == nil {
		return false
	}
	for _, node := range t.Tree.Root.Nodes {
		if node.Type() == parse.NodeAction {
			return true
		}
	}
	return false
}

type ErrParse struct {
	filename string
	raw error
}

func NewErrParse(filename string, err error) *ErrParse {
	return &ErrParse{filename, err}
}

func (e *ErrParse) NiceError() string {
	return fmt.Sprintf("Failed to parse a template: %s. Error: %s", e.filename, e.raw.Error())
}

type ErrExec struct {
	filename string
	ctx string
	raw error
}

func NewErrExec(filename, ctx string, err error) *ErrExec {
	return &ErrExec{filename, ctx, err}
}

func (e *ErrExec) NiceError() string {
	return fmt.Sprintf("Failed to execute a template:\n%s [%s].\nError: %s", e.filename, e.ctx, e.raw)
}

func ExecToString(t *template.Template, dot interface{}) (string, error) {
	b := strings.Builder{}
	err := t.Execute(&b, dot)
	return b.String(), err
}

func New(name string) *template.Template {
	return template.New(name).Funcs(template.FuncMap{
		"to_yaml": ToYaml,
	})
}

var ToYaml = func(val interface{}) (string, error) {
	b := strings.Builder{}
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	err := encoder.Encode(val)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}