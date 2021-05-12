package go_command_chain

import (
	"fmt"
	"io"
	"os/exec"
)

type chain struct {
	cmdDescriptors []cmdDescriptor

	stdIn  io.Reader
	stdOut io.Writer
	stdErr io.Writer
}

type ChainBuilder interface {
	Join(name string, args ...string) CommandBuilder
	JoinCmd(cmd *exec.Cmd) CommandBuilder

	Finalize() FinalizedBuilder
}

type CommandBuilder interface {
	ChainBuilder

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

type cmdDescriptor struct {
	command  *exec.Cmd
	outForks []io.Writer
	errForks []io.Writer
}

func Builder() FirstCommandBuilder {
	return &chain{}
}

func (c *chain) JoinCmd(cmd *exec.Cmd) CommandBuilder {
	if cmd != nil {
		c.cmdDescriptors = append(c.cmdDescriptors, cmdDescriptor{
			command: cmd,
		})
	}

	return c
}

func (c *chain) Join(name string, args ...string) CommandBuilder {
	return c.JoinCmd(exec.Command(name, args...))
}

func (c *chain) WithOutputForks(targets ...io.Writer) CommandBuilder {
	c.cmdDescriptors[len(c.cmdDescriptors)-1].outForks = targets
	return c
}

func (c *chain) WithErrorForks(targets ...io.Writer) CommandBuilder {
	c.cmdDescriptors[len(c.cmdDescriptors)-1].errForks = targets
	return c
}

func (c *chain) WithInput(r io.Reader) ChainBuilder {
	c.cmdDescriptors[0].command.Stdin = r
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
	return c
}

func (c *chain) build() error {
	//link each command with pipelines
	//
	//stdIn -> cmd1(stdIn)
	//	cmd1(stdOut/stdErr) -> cmd2(stdIn)
	//	cmd2(stdOut/stdErr) -> cmd3(stdIn)
	//cmd3(stdOut/stdErr) -> stdOut/stdErr

	for i := 1; i < len(c.cmdDescriptors); i++ {
		var err error
		c.cmdDescriptors[i].command.Stdin, err = c.cmdDescriptors[i-1].command.StdoutPipe()
		if err != nil {
			return fmt.Errorf("unable to chain command stream: %w", err)
		}
	}
	return nil
}

func (c *chain) Run() error {
	err := c.build()
	if err != nil {
		return err
	}

	//we have to start all commands (non blocking!)
	for _, cmdDescriptor := range c.cmdDescriptors {
		err := cmdDescriptor.command.Start()
		if err != nil {
			return fmt.Errorf("failed to start command: %w", err)
		}
	}

	runErrors := RunErrors{
		errors: make([]error, len(c.cmdDescriptors)),
	}
	hasError := false

	for i, cmdDescriptor := range c.cmdDescriptors {
		runErrors.errors[i] = cmdDescriptor.command.Wait()
		if runErrors.errors[i] != nil {
			hasError = true
		}
	}

	if hasError {
		return runErrors
	}
	return nil
}
