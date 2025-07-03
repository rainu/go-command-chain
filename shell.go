package cmdchain

import (
	"fmt"
	"io"
	"mvdan.cc/sh/v3/syntax"
	"strings"
)

func FromShell(command string, sources ...io.Reader) (FinalizedBuilder, error) {
	sp, err := newShellParser(command, sources)
	if err != nil {
		return nil, err
	}
	return sp.Parse()
}

type shellParser struct {
	program *syntax.File
	chain   *chain
}

func newShellParser(command string, sources []io.Reader) (*shellParser, error) {
	result := &shellParser{}
	if len(sources) == 0 {
		result.chain = Builder().(*chain)
	} else {
		result.chain = Builder().WithInput(sources...).(*chain)
	}

	var err error

	result.program, err = syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, fmt.Errorf("error parsing shell command: %w", err)
	}

	return result, nil
}

func (s *shellParser) Parse() (FinalizedBuilder, error) {
	if len(s.program.Stmts) == 0 {
		return nil, fmt.Errorf("no statements")
	}
	if len(s.program.Stmts) > 1 {
		return nil, fmt.Errorf("multiple statements are not supported, found %d statements", len(s.program.Stmts))
	}

	err := s.handleCommand(s.program.Stmts[0].Cmd)
	if err != nil {
		return nil, err
	}

	return s.chain.Finalize(), nil
}

func (s *shellParser) handleCommand(cmd syntax.Command) error {
	switch c := cmd.(type) {
	case *syntax.CallExpr:
		return s.handleCall(c)
	case *syntax.BinaryCmd:
		return s.handleBinary(c)
	default:
		return errorWithPos(c, "unsupported command")
	}
}

func (s *shellParser) handleCall(c *syntax.CallExpr) error {
	commandName, arguments, err := s.extractCommandAndArgs(c.Args)
	if err != nil {
		return errorWithPos(c, "error extracting command and arguments", err)
	}

	s.chain = s.chain.Join(commandName, arguments...).(*chain)
	return s.handleAssigns(c.Assigns)
}

func (s *shellParser) handleAssigns(assigns []*syntax.Assign) error {
	var env []string

	for _, assign := range assigns {
		if assign.Value == nil && assign.Array == nil && assign.Index == nil {
			// This is a simple assignment without value, e.g., `VAR=`
			env = append(env, fmt.Sprintf("%s=", assign.Name.Value))
		} else if assign.Value != nil {
			value, err := s.convertWord(assign.Value)
			if err != nil {
				return errorWithPos(assign, "error converting assignment value", err)
			}
			env = append(env, fmt.Sprintf("%s=%s", assign.Name.Value, value))
		} else {
			return errorWithPos(assign, "unsupported assignment")
		}
	}

	s.chain.WithAdditionalEnvironmentPairs(env...)
	return nil
}

func (s *shellParser) handleBinary(b *syntax.BinaryCmd) error {
	if err := s.handleCommand(b.X.Cmd); err != nil {
		return err
	}

	switch b.Op {
	case syntax.Pipe: // |
	case syntax.PipeAll: // |&
		s.chain = s.chain.ForwardError().(*chain)
	default:
		return errorWithPos(b, fmt.Sprintf("unsupported binary operator at '%s'", b.OpPos.String()))
	}

	return s.handleCommand(b.Y.Cmd)
}

func (s *shellParser) extractCommandAndArgs(words []*syntax.Word) (commandName string, arguments []string, err error) {
	for i := range words {
		if i == 0 {
			commandName, err = s.convertWord(words[i])
			if err != nil {
				return
			}
		} else {
			var argument string
			argument, err = s.convertWord(words[i])
			if err != nil {
				return
			}

			arguments = append(arguments, argument)
		}
	}

	return
}

func (s *shellParser) convertWord(word *syntax.Word) (string, error) {
	if word == nil {
		return "", nil
	}

	result := word.Lit()
	if result != "" {
		return result, nil
	}

	return s.convertWordParts(word.Parts)
}

func (s *shellParser) convertWordParts(parts []syntax.WordPart) (result string, err error) {
	for i := range parts {
		switch part := parts[i].(type) {
		case *syntax.Lit:
			result += part.Value
		case *syntax.SglQuoted:
			result += part.Value
		case *syntax.DblQuoted:
			var r string
			r, err = s.convertWordParts(part.Parts)
			if err != nil {
				return
			}

			result += r
		default:
			err = errorWithPos(part, "unsupported word")
			return
		}
	}
	return
}
