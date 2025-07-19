package cmdchain

import (
	"context"
	"fmt"
	"io"
	"mvdan.cc/sh/v3/syntax"
	"os"
)

type shellParser struct {
	program *syntax.File
	ctx     context.Context
	chain   *chain
	actions []func(CommandBuilder)
}

func (s *shellParser) applyActions() {
	for _, action := range s.actions {
		action(s.chain)
	}
}

func (s *shellParser) Parse() error {
	if len(s.program.Stmts) == 0 {
		return fmt.Errorf("no statements")
	}
	if len(s.program.Stmts) > 1 {
		return fmt.Errorf("multiple statements are not supported, found %d statements", len(s.program.Stmts))
	}
	if s.program.Stmts[0].Background {
		return fmt.Errorf("background execution is not supported")
	}

	err := s.handleCommand(s.program.Stmts[0].Cmd, s.program.Stmts[0].Redirs)
	if err != nil {
		return err
	}

	return nil
}

func (s *shellParser) handleCommand(cmd syntax.Command, redirs []*syntax.Redirect) error {
	switch c := cmd.(type) {
	case *syntax.CallExpr:
		return s.handleCall(c, redirs)
	case *syntax.BinaryCmd:
		return s.handleBinary(c)
	default:
		return errorWithPos(c, "unsupported command")
	}
}

func (s *shellParser) handleCall(c *syntax.CallExpr, redirs []*syntax.Redirect) error {
	commandName, arguments, err := s.extractCommandAndArgs(c.Args)
	if err != nil {
		return errorWithPos(c, "error extracting command and arguments", err)
	}

	if s.ctx == nil {
		s.chain = s.chain.Join(commandName, arguments...).(*chain)
	} else {
		s.chain = s.chain.JoinWithContext(s.ctx, commandName, arguments...).(*chain)
	}

	err = s.handleAssigns(c.Assigns)
	if err != nil {
		return err
	}

	err = s.handleRedirects(redirs)
	if err != nil {
		return err
	}

	s.applyActions()
	return nil
}

func (s *shellParser) handleRedirects(redirs []*syntax.Redirect) (err error) {
	var outputStreams []io.Writer
	var errorStreams []io.Writer

	for _, redir := range redirs {
		switch redir.Op {
		case syntax.RdrAll: // &>
		case syntax.AppAll: // &>>
		case syntax.RdrOut: // >
		case syntax.AppOut: // >>
		default:
			return errorWithPos(redir, fmt.Sprintf("unsupported redirection operator '%s'", redir.Op.String()))
		}

		var targetFile *lazyFile
		targetFile, err = s.setupStream(redir)
		if err != nil {
			return err
		}

		// register file-hook to ensure the file is closed after command execution
		s.chain.addHook(targetFile)

		if redir.Op == syntax.RdrAll || redir.Op == syntax.AppAll {
			errorStreams = append(errorStreams, targetFile)
			outputStreams = append(outputStreams, targetFile)
		} else if redir.N != nil && redir.N.Value == "2" {
			errorStreams = append(errorStreams, targetFile)
		} else {
			outputStreams = append(outputStreams, targetFile)
		}
	}

	s.chain.WithOutputForks(outputStreams...)
	s.chain.WithErrorForks(errorStreams...)

	return nil
}

func (s *shellParser) setupStream(redir *syntax.Redirect) (*lazyFile, error) {
	target, err := s.convertWord(redir.Word)
	if err != nil {
		return nil, errorWithPos(redir, "error converting output redirection target", err)
	}
	if target == "" {
		return nil, errorWithPos(redir, "missing output redirection target")
	}

	flag := os.O_WRONLY | os.O_CREATE

	switch redir.Op {
	case syntax.RdrAll:
		fallthrough
	case syntax.RdrOut:
		flag |= os.O_TRUNC
	case syntax.AppAll:
		fallthrough
	case syntax.AppOut:
		flag |= os.O_APPEND
	default:
	}

	return newLazyFile(target, flag, 0644), nil
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
	if err := s.handleCommand(b.X.Cmd, b.X.Redirs); err != nil {
		return err
	}

	switch b.Op {
	case syntax.Pipe: // |
	case syntax.PipeAll: // |&
		s.chain = s.chain.ForwardError().(*chain)
	default:
		return errorWithPos(b, fmt.Sprintf("unsupported binary operator '%s' at '%s'", b.Op.String(), b.OpPos.String()))
	}

	return s.handleCommand(b.Y.Cmd, b.Y.Redirs)
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
