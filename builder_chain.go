package command_chain

import (
	"context"
	"io"
	"os/exec"
	"sync"
)

type chain struct {
	cmdDescriptors []cmdDescriptor

	input    io.Reader
	inputErr error

	buildErrors  MultipleErrors
	streamErrors MultipleErrors

	streamRoutinesWg sync.WaitGroup
}

type cmdDescriptor struct {
	command *exec.Cmd
	outToIn bool
	errToIn bool
	outFork io.Writer
	errFork io.Writer
}

func Builder() FirstCommandBuilder {
	return &chain{
		buildErrors:      buildErrors(),
		streamErrors:     streamErrors(),
		streamRoutinesWg: sync.WaitGroup{},
	}
}

func (c *chain) WithInput(sources ...io.Reader) ChainBuilder {
	if len(sources) == 1 {
		c.input = sources[0]
	} else if len(sources) > 1 {
		c.input, c.inputErr = c.combineStream(sources...)
	}

	return c
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

func (c *chain) JoinWithContext(ctx context.Context, name string, args ...string) CommandBuilder {
	return c.JoinCmd(exec.CommandContext(ctx, name, args...))
}

func (c *chain) Finalize() FinalizedBuilder {
	if len(c.cmdDescriptors) > 0 {
		c.cmdDescriptors[0].command.Stdin = c.input
		if c.streamErrors.Errors()[0] == nil {
			c.streamErrors.setError(0, c.inputErr)
		}
	}
	return c
}
