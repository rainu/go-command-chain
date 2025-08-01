package cmdchain

import (
	"context"
	"io"
	"os/exec"
	"sync"
)

type chain struct {
	cmdDescriptors []cmdDescriptor
	inputs         []io.Reader
	buildErrors    MultipleErrors
	streamErrors   MultipleErrors

	streamRoutinesWg sync.WaitGroup
	errorChecker     ErrorChecker

	hooks []hook
}

type cmdDescriptor struct {
	command        *exec.Cmd
	outToIn        bool
	errToIn        bool
	outFork        io.Writer
	errFork        io.Writer
	commandApplier []CommandApplier
	errorChecker   ErrorChecker

	inputStreams  []io.Reader
	outputStreams []io.Writer
	errorStreams  []io.Writer
}

// Builder creates a new command chain builder. This build flow will configure
// the commands more or less instantaneously. If any error occurs while building
// the chain you will receive them when you finally call Run of this chain.
func Builder() FirstCommandBuilder {
	return &chain{
		buildErrors:      buildErrors(),
		streamErrors:     streamErrors(),
		streamRoutinesWg: sync.WaitGroup{},
	}
}

func (c *chain) WithInput(sources ...io.Reader) ChainBuilder {
	c.inputs = sources
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
	if len(c.cmdDescriptors) == 0 {
		return c
	}

	firstCmdDesc := &(c.cmdDescriptors[0])

	is := firstCmdDesc.inputStreams
	firstCmdDesc.inputStreams = append([]io.Reader{}, c.inputs...)
	firstCmdDesc.inputStreams = append(firstCmdDesc.inputStreams, is...)

	if len(c.inputs) == 1 {
		firstCmdDesc.command.Stdin = c.inputs[0]
	} else if len(c.inputs) > 1 {
		var err error
		firstCmdDesc.command.Stdin, err = c.combineStreamForCommand(0, c.inputs...)
		if c.streamErrors.Errors()[0] == nil {
			c.streamErrors.setError(0, err)
		}
	}

	lastCmdDesc := &(c.cmdDescriptors[len(c.cmdDescriptors)-1])
	if lastCmdDesc.outFork != nil {
		lastCmdDesc.command.Stdout = lastCmdDesc.outFork
	}
	if lastCmdDesc.errFork != nil {
		lastCmdDesc.command.Stderr = lastCmdDesc.errFork
	}

	return c
}
