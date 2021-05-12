package go_command_chain

import (
	"fmt"
	"strings"
)

type RunErrors struct {
	errors []error
}

func (e RunErrors) Errors() []error {
	return e.errors
}

func (e RunErrors) Error() string {
	sb := strings.Builder{}

	sb.WriteString("one ore more command has returned an error: [")
	for i, err := range e.errors {
		sb.WriteString(fmt.Sprintf("%d - %s", i, err.Error()))
		if i+1 != len(e.errors) {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("]")

	return sb.String()
}
