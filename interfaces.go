package command_chain

import (
	"context"
	"io"
	"os/exec"
)

type ChainBuilder interface {
	Join(name string, args ...string) CommandBuilder
	JoinCmd(cmd *exec.Cmd) CommandBuilder
	JoinWithContext(ctx context.Context, name string, args ...string) CommandBuilder

	Finalize() FinalizedBuilder
}

type FirstCommandBuilder interface {
	CommandBuilder

	WithInput(r io.Reader) ChainBuilder
}

type CommandBuilder interface {
	ChainBuilder

	ForwardError() CommandBuilder
	BlockingOutput() CommandBuilder
	WithOutputForks(targets ...io.Writer) CommandBuilder
	WithErrorForks(targets ...io.Writer) CommandBuilder
}

type FinalizedBuilder interface {
	WithOutput(w io.Writer) FinalizedBuilder
	WithError(w io.Writer) FinalizedBuilder

	Run() error
}
