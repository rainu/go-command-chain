package cmdchain

import (
	"context"
	"fmt"
	"io"
	"mvdan.cc/sh/v3/syntax"
	"os/exec"
	"strings"
)

func (c *chain) JoinShellCmd(command string) CommandBuilder {
	return &shellChain{
		command: command,
		chain:   c,
	}
}

func (c *chain) JoinShellCmdWithContext(ctx context.Context, command string) CommandBuilder {
	return &shellChain{
		ctx:     ctx,
		command: command,
		chain:   c,
	}
}

type shellChain struct {
	ctx     context.Context
	command string

	actions []func(CommandBuilder)
	chain   *chain
}

var buildShellChain = func(s *shellChain) CommandBuilder {
	var err error
	defer func() {
		if err != nil {
			s.chain.buildErrors.addError(fmt.Errorf("error parsing shell command: %w", err))
		}
	}()

	parser := &shellParser{
		chain:   s.chain,
		ctx:     s.ctx,
		actions: s.actions,
	}

	parser.program, err = syntax.NewParser().Parse(strings.NewReader(s.command), "")
	if err != nil {
		return s.chain
	}

	err = parser.Parse()

	return s.chain
}

////
// Before joining the next command, here we build the current shell-command and apply all collected actions.
////

func (s *shellChain) Join(name string, args ...string) CommandBuilder {
	return buildShellChain(s).Join(name, args...)
}

func (s *shellChain) JoinCmd(cmd *exec.Cmd) CommandBuilder {
	return buildShellChain(s).JoinCmd(cmd)
}

func (s *shellChain) JoinWithContext(ctx context.Context, name string, args ...string) CommandBuilder {
	return buildShellChain(s).JoinWithContext(ctx, name, args...)
}

func (s *shellChain) JoinShellCmd(command string) CommandBuilder {
	return buildShellChain(s).JoinShellCmd(command)
}

func (s *shellChain) JoinShellCmdWithContext(ctx context.Context, command string) CommandBuilder {
	return buildShellChain(s).JoinShellCmdWithContext(ctx, command)
}

func (s *shellChain) Finalize() FinalizedBuilder {
	return buildShellChain(s).Finalize()
}

////
// Here we have to "collect" the actions which must be applied BEFORE the next command is joined.
////

func (s *shellChain) Apply(applier CommandApplier) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.Apply(applier)
	})
	return s
}

func (s *shellChain) ApplyBeforeStart(applier CommandApplier) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.ApplyBeforeStart(applier)
	})
	return s
}

func (s *shellChain) ForwardError() CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.ForwardError()
	})
	return s
}

func (s *shellChain) DiscardStdOut() CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.DiscardStdOut()
	})
	return s
}

func (s *shellChain) WithOutputForks(targets ...io.Writer) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithOutputForks(targets...)
	})
	return s
}

func (s *shellChain) WithAdditionalOutputForks(targets ...io.Writer) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithAdditionalOutputForks(targets...)
	})
	return s
}

func (s *shellChain) WithErrorForks(targets ...io.Writer) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithErrorForks(targets...)
	})
	return s
}

func (s *shellChain) WithAdditionalErrorForks(targets ...io.Writer) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithAdditionalErrorForks(targets...)
	})
	return s
}

func (s *shellChain) WithInjections(sources ...io.Reader) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithInjections(sources...)
	})
	return s
}

func (s *shellChain) WithEmptyEnvironment() CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithEmptyEnvironment()
	})
	return s
}

func (s *shellChain) WithEnvironment(envMap ...interface{}) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithEnvironment(envMap...)
	})
	return s
}

func (s *shellChain) WithEnvironmentMap(envMap map[interface{}]interface{}) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithEnvironmentMap(envMap)
	})
	return s
}

func (s *shellChain) WithEnvironmentPairs(envMap ...string) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithEnvironmentPairs(envMap...)
	})
	return s
}

func (s *shellChain) WithAdditionalEnvironment(envMap ...interface{}) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithAdditionalEnvironment(envMap...)
	})
	return s
}

func (s *shellChain) WithAdditionalEnvironmentMap(envMap map[interface{}]interface{}) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithAdditionalEnvironmentMap(envMap)
	})
	return s
}

func (s *shellChain) WithAdditionalEnvironmentPairs(envMap ...string) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithAdditionalEnvironmentPairs(envMap...)
	})
	return s
}

func (s *shellChain) WithWorkingDirectory(workingDir string) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithWorkingDirectory(workingDir)
	})
	return s
}

func (s *shellChain) WithErrorChecker(checker ErrorChecker) CommandBuilder {
	s.actions = append(s.actions, func(c CommandBuilder) {
		c.WithErrorChecker(checker)
	})
	return s
}
