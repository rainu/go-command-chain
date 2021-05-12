package go_command_chain

import (
	"fmt"
	"io"
	"os/exec"
)

type chain struct {
	cmdDescriptors []cmdDescriptor
	buildErrors    BuildErrors
}

type cmdDescriptor struct {
	command  *exec.Cmd
	outForks []io.Writer
	errForks []io.Writer
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

func Builder() FirstCommandBuilder {
	return &chain{}
}

func (c *chain) JoinCmd(cmd *exec.Cmd) CommandBuilder {
	if cmd == nil {
		return c
	}

	c.cmdDescriptors = append(c.cmdDescriptors, cmdDescriptor{
		command: cmd,
	})

	if len(c.cmdDescriptors) > 1 {
		var err error

		//link this command's input with the previous command's output (cmd1 -> cmd2)
		cmd.Stdin, err = c.cmdDescriptors[len(c.cmdDescriptors)-2].command.StdoutPipe()
		c.buildErrors.addError(err)
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

	runErrors := RunErrors{}
	for _, cmdDescriptor := range c.cmdDescriptors {
		runErrors.addError(cmdDescriptor.command.Wait())
	}

	if runErrors.hasError {
		return runErrors
	}
	return nil
}
