package varmap

import (
	"strings"
)

type Path struct {
	Elements []string
}

func NewPath() *Path {
	return &Path{
		Elements: make([]string, 0, 4),
	}
}

func (p *Path) CopyJoin(els ...string) *Path {
	np := &Path{
		Elements: make([]string, 0, len(p.Elements) + len(els)),
	}
	np.Elements = append(np.Elements, p.Elements...)
	np.Elements = append(np.Elements, els...)
	return np
}

func (p *Path) Join(els ...string) *Path {
	p.Elements = append(p.Elements, els...)
	return p
}

func (p *Path) String() string {
	return strings.Join(p.Elements, ".")
}

func charsCount(els []string) int {
	sum := 0
	for i, el := range els {
		sum += len(el)
		if i != len(els)-1 {
			sum++
		}
	}
	return sum
}

func (p *Path) CharsCount() int {
	return charsCount(p.Elements)
}