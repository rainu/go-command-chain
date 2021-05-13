package go_command_chain

import (
	"fmt"
	"strings"
)

type MultipleErrors struct {
	errorMessage string
	errors       []error
	hasError     bool
}

func (e MultipleErrors) Errors() []error {
	return e.errors
}

func (e MultipleErrors) Error() string {
	sb := strings.Builder{}

	sb.WriteString(e.errorMessage)
	sb.WriteString(": [")
	for i, err := range e.errors {
		sb.WriteString(fmt.Sprintf("%d - ", i))
		if err != nil {
			sb.WriteString(err.Error())
		}

		if i+1 != len(e.errors) {
			sb.WriteString("; ")
		}
	}
	sb.WriteString("]")

	return sb.String()
}

func (e *MultipleErrors) addError(err error) {
	e.errors = append(e.errors, err)
	if err != nil {
		if mError, ok := err.(MultipleErrors); ok {
			e.hasError = mError.hasError
		} else {
			e.hasError = true
		}
	}
}

func (e *MultipleErrors) setError(i int, err error) {
	e.errors[i] = err
	if err != nil {
		if mError, ok := err.(MultipleErrors); ok {
			e.hasError = mError.hasError
		} else {
			e.hasError = true
		}
	}
}

func runErrors() MultipleErrors {
	return MultipleErrors{
		errorMessage: "one or more command has returned an error",
	}
}

func buildErrors() MultipleErrors {
	return MultipleErrors{
		errorMessage: "one or more chain build errors occurred",
	}
}

func streamErrors() MultipleErrors {
	return MultipleErrors{
		errorMessage: "one or more command stream copies failed",
	}
}
