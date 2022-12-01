package varmap

import "fmt"

type ResolutionError struct {
	Path    Path
	Sources [2]string
}

func (e *ResolutionError) Error() string {
	return fmt.Sprintf(
		"resolution conflict on variable %s. First defined in: %s. Second defined in: %s",
		e.Path.String(),
		e.Sources[0],
		e.Sources[1],
	)
}
