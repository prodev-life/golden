package rerrors

import (
	"fmt"
	"strings"
)

type NiceError interface {
	NiceError() string
}

type ErrIo struct {
	file string
	ctx string
	raw error
}

func NewErrIo(file, ctx string, raw error) *ErrIo {
	return &ErrIo{file, ctx, raw}
}

func (e *ErrIo) NiceError() string {
	return fmt.Sprintf(
		"Failed io on file: %s. Context: %s. Internal error: %s",
		e.file, e.ctx, e.raw,
	)
}

type ErrDuplicate struct {
	dupName string
	typ string
	occurrances []string
}

func NewErrDuplicate(dupName, typ string, occurrances ...string) *ErrDuplicate {
	return &ErrDuplicate{dupName, typ, occurrances}
}

func (e *ErrDuplicate) NiceError() string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("Duplicate %s: %s", e.typ, e.dupName))
	if len(e.occurrances) > 0 {
		b.WriteString("\nEncountered in:")
		for _, occ := range e.occurrances {
			b.WriteString(fmt.Sprintf("\n\t%s",occ))
		}
	}
	return b.String()
}

func (e *ErrDuplicate) Error() string {
	return e.NiceError()
}

type ErrString struct {
	msg string
}

func NewErrStringf(format string, args... interface{}) *ErrString {
	return &ErrString{fmt.Sprintf(format, args...)}
}

func (e *ErrString) NiceError() string {
	return e.msg
}