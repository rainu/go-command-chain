package cmdchain

import (
	"fmt"
	"io"
	"mvdan.cc/sh/v3/syntax"
	"os"
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

	err := s.handleCommand(s.program.Stmts[0].Cmd, s.program.Stmts[0].Redirs)
	if err != nil {
		return nil, err
	}

	return s.chain.Finalize(), nil
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

	s.chain = s.chain.Join(commandName, arguments...).(*chain)

	err = s.handleAssigns(c.Assigns)
	if err != nil {
		return err
	}

	return s.handleRedirects(redirs)
}

func (s *shellParser) handleRedirects(redirs []*syntax.Redirect) (err error) {
	var outputStreams []io.Writer
	var errorStreams []io.Writer
	var files []*os.File

	// in case of any error, we have to ensure that all opened files are closed
	defer func() {
		if err != nil {
			for _, file := range files {
				file.Close()
			}
		}
	}()

	for _, redir := range redirs {
		switch redir.Op {
		case syntax.RdrAll: // &>
		case syntax.AppAll: // &>>
		case syntax.RdrOut: // >
		case syntax.AppOut: // >>
		default:
			return errorWithPos(redir, fmt.Sprintf("unsupported redirection operator '%s'", redir.Op.String()))
		}

		var targetFile *os.File
		targetFile, err = s.setupStream(redir)
		if err != nil {
			return err
		}
		files = append(files, targetFile)

		if redir.Op != syntax.RdrOut && redir.Op != syntax.AppOut {
			errorStreams = append(errorStreams, targetFile)
		}
		outputStreams = append(outputStreams, targetFile)
	}

	s.chain.WithOutputForks(outputStreams...)
	s.chain.WithErrorForks(errorStreams...)

	return nil
}

func (s *shellParser) setupStream(redir *syntax.Redirect) (*os.File, error) {
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

	//TODO: must be closed after command execution; Must be "reopen" if command run again
	// maybe a "lazy file" would be a good idea:
	// * only open when first write (this would prevent opening files that are never used)
	// * close when command execution finished
	f, err := os.OpenFile(target, flag, 0644)
	if err != nil {
		return nil, errorWithPos(redir, fmt.Sprintf("error opening output file '%s'", target), err)
	}
	return f, nil
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
