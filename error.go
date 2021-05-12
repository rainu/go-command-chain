package go_command_chain

import (
	"fmt"
	"strings"
)

type BuildErrors struct {
	errors   []error
	hasError bool
}

func (e BuildErrors) Errors() []error {
	return e.errors
}

func (e BuildErrors) Error() string {
	return errorString("one or more chain build errors occurred", e.errors)
}

func (e BuildErrors) addError(err error) {
	e.errors = append(e.errors, err)
	if err != nil {
		e.hasError = true
	}
}

type RunErrors struct {
	errors   []error
	hasError bool
}

func (e RunErrors) Errors() []error {
	return e.errors
}

func (e RunErrors) Error() string {
	return errorString("one ore more command has returned an error", e.errors)
}

func (e RunErrors) addError(err error) {
	e.errors = append(e.errors, err)
	if err != nil {
		e.hasError = true
	}
}

func errorString(msg string, errors []error) string {
	sb := strings.Builder{}

	sb.WriteString(msg)
	sb.WriteString(": [")
	for i, err := range errors {
		sb.WriteString(fmt.Sprintf("%d - %s", i, err.Error()))
		if i+1 != len(errors) {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("]")

	return sb.String()
}
