package tasker

import (
	"fmt"
)


type CycleError [][]string

func (ce CycleError) Error() string {
	msg := "tasker: "
	if len(ce) > 1 {
		msg += "cycles"
	} else {
		msg += "cycle"
	}
	msg += " detected: "
	for i, c := range ce {
		for j, e := range c {
			msg += e
			if j < len(c) - 1 {
				msg += " -> "
			}
		}
		if i < len(ce) - 1 {
			msg += ", "
		}
	}
	return msg
}

type DepNotFoundError struct {
	v string
	w string
}

func NewDepNotFoundError(v, w string) *DepNotFoundError {
	return &DepNotFoundError{v, w}
}

func (dnfe *DepNotFoundError) Error() string {
	return fmt.Sprintf("tasker: %s not found, required by %s", dnfe.w, dnfe.v)
}
