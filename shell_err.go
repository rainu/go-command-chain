package cmdchain

import (
	"errors"
	"fmt"
	"mvdan.cc/sh/v3/syntax"
)

type positionProvider interface {
	Pos() syntax.Pos
	End() syntax.Pos
}

type errorWithPosition struct {
	Position positionProvider
	Message  string
}

func (e *errorWithPosition) Error() string {
	return fmt.Sprintf("[%s - %s] %s", e.Position.Pos().String(), e.Position.End().String(), e.Message)
}

func errorWithPos(pos positionProvider, message string, cause ...error) error {
	err := &errorWithPosition{
		Position: pos,
		Message:  message,
	}

	if len(cause) > 0 {
		return errors.Join(err, cause[0])
	}

	return err
}
