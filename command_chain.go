package go_command_chain

import (
	"fmt"
	"io"
	"os/exec"
)

type chain struct {
	cmdDescriptors []cmdDescriptor
	input          io.Reader
	buildErrors    MultipleErrors
	streamErrors   MultipleErrors
}

type cmdDescriptor struct {
	command *exec.Cmd
	outToIn bool
	errToIn bool
	outFork io.Writer
	errFork io.Writer
}

type ChainBuilder interface {
	Join(name string, args ...string) CommandBuilder
	JoinCmd(cmd *exec.Cmd) CommandBuilder

	Finalize() FinalizedBuilder
}

type CommandBuilder interface {
	ChainBuilder

	ForwardError() CommandBuilder
	BlockingOutput() CommandBuilder
	WithOutputForks(targets ...io.Writer) CommandBuilder
	WithErrorForks(targets ...io.Writer) CommandBuilder
}

type FirstCommandBuilder interface {
	CommandBuilder

	WithInput(r io.Reader) ChainBuilder
}

type FinalizedBuilder interface {
	WithOutput(w io.Writer) FinalizedBuilder
	WithError(w io.Writer) FinalizedBuilder

	Run() error
}

func Builder() FirstCommandBuilder {
	return &chain{
		buildErrors:  buildErrors(),
		streamErrors: streamErrors(),
	}
}

func (c *chain) JoinCmd(cmd *exec.Cmd) CommandBuilder {
	if cmd == nil {
		return c
	}

	c.cmdDescriptors = append(c.cmdDescriptors, cmdDescriptor{
		command: cmd,
		outToIn: true,
	})
	c.streamErrors.addError(nil)

	if len(c.cmdDescriptors) > 1 {
		c.linkStreams(cmd)
	}

	return c
}

func (c *chain) Join(name string, args ...string) CommandBuilder {
	return c.JoinCmd(exec.Command(name, args...))
}

func (c *chain) ForwardError() CommandBuilder {
	c.cmdDescriptors[len(c.cmdDescriptors)-1].errToIn = true
	return c
}

func (c *chain) BlockingOutput() CommandBuilder {
	c.cmdDescriptors[len(c.cmdDescriptors)-1].outToIn = false
	return c
}

func (c *chain) WithOutputForks(targets ...io.Writer) CommandBuilder {
	if len(targets) > 1 {
		c.cmdDescriptors[len(c.cmdDescriptors)-1].outFork = io.MultiWriter(targets...)
	} else if len(targets) == 1 {
		c.cmdDescriptors[len(c.cmdDescriptors)-1].outFork = targets[0]
	}

	return c
}

func (c *chain) WithErrorForks(targets ...io.Writer) CommandBuilder {
	if len(targets) > 1 {
		c.cmdDescriptors[len(c.cmdDescriptors)-1].errFork = io.MultiWriter(targets...)
	} else if len(targets) == 1 {
		c.cmdDescriptors[len(c.cmdDescriptors)-1].errFork = targets[0]
	}
	return c
}

func (c *chain) WithInput(r io.Reader) ChainBuilder {
	c.input = r
	return c
}

func (c *chain) WithOutput(w io.Writer) FinalizedBuilder {
	c.cmdDescriptors[len(c.cmdDescriptors)-1].command.Stdout = w
	return c
}

func (c *chain) WithError(w io.Writer) FinalizedBuilder {
	c.cmdDescriptors[len(c.cmdDescriptors)-1].command.Stderr = w
	return c
}

func (c *chain) Finalize() FinalizedBuilder {
	if len(c.cmdDescriptors) > 0 {
		c.cmdDescriptors[0].command.Stdin = c.input
	}
	return c
}

func (c *chain) Run() error {
	if c.buildErrors.hasError {
		return c.buildErrors
	}

	//we have to start all commands (non blocking!)
	for _, cmdDescriptor := range c.cmdDescriptors {
		err := cmdDescriptor.command.Start()
		if err != nil {
			return fmt.Errorf("failed to start command: %w", err)
		}
	}

	runErrors := runErrors()
	for _, cmdDescriptor := range c.cmdDescriptors {
		runErrors.addError(cmdDescriptor.command.Wait())
	}

	switch {
	case runErrors.hasError && c.streamErrors.hasError:
		return MultipleErrors{
			errorMessage: "run and stream errors occurred",
			errors:       []error{runErrors, c.streamErrors},
			hasError:     true,
		}
	case runErrors.hasError:
		return runErrors
	case c.streamErrors.hasError:
		return c.streamErrors
	default:
		return nil
	}
}
