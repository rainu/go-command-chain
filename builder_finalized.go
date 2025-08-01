package cmdchain

import (
	"bytes"
	"fmt"
	"io"
)

func (c *chain) WithOutput(targets ...io.Writer) FinalizedBuilder {
	cmdDesc := &(c.cmdDescriptors[len(c.cmdDescriptors)-1])
	cmdDesc.outputStreams = targets

	if len(targets) == 1 {
		cmdDesc.command.Stdout = targets[0]
	} else if len(targets) > 1 {
		cmdDesc.command.Stdout = io.MultiWriter(targets...)
	}

	return c
}

func (c *chain) WithAdditionalOutput(targets ...io.Writer) FinalizedBuilder {
	cmdDesc := &(c.cmdDescriptors[len(c.cmdDescriptors)-1])
	cmdDesc.outputStreams = append(cmdDesc.outputStreams, targets...)

	if len(cmdDesc.outputStreams) == 1 {
		cmdDesc.command.Stdout = cmdDesc.outputStreams[0]
	} else if len(cmdDesc.outputStreams) > 1 {
		cmdDesc.command.Stdout = io.MultiWriter(cmdDesc.outputStreams...)
	}

	return c
}

func (c *chain) WithError(targets ...io.Writer) FinalizedBuilder {
	cmdDesc := &(c.cmdDescriptors[len(c.cmdDescriptors)-1])
	cmdDesc.errorStreams = targets

	if len(targets) == 1 {
		cmdDesc.command.Stderr = targets[0]
	} else if len(targets) > 1 {
		cmdDesc.command.Stderr = io.MultiWriter(targets...)
	}

	return c
}

func (c *chain) WithAdditionalError(targets ...io.Writer) FinalizedBuilder {
	cmdDesc := &(c.cmdDescriptors[len(c.cmdDescriptors)-1])
	cmdDesc.errorStreams = append(cmdDesc.errorStreams, targets...)

	if len(cmdDesc.errorStreams) == 1 {
		cmdDesc.command.Stderr = cmdDesc.errorStreams[0]
	} else if len(cmdDesc.errorStreams) > 1 {
		cmdDesc.command.Stderr = io.MultiWriter(cmdDesc.errorStreams...)
	}

	return c
}

func (c *chain) WithGlobalErrorChecker(errorChecker ErrorChecker) FinalizedBuilder {
	c.errorChecker = errorChecker
	return c
}

func (c *chain) RunAndGet() (string, string, error) {
	streamOut := &bytes.Buffer{}
	streamErr := &bytes.Buffer{}

	err := c.WithAdditionalOutput(streamOut).WithAdditionalError(streamErr).Run()

	return streamOut.String(), streamErr.String(), err
}

func (c *chain) Run() error {
	if c.buildErrors.hasError {
		return c.buildErrors
	}

	c.executeBeforeRunHooks()
	defer c.executeAfterRunHooks()

	//we have to start all commands (non blocking!)
	for cmdIndex, cmdDescriptor := range c.cmdDescriptors {
		for _, applier := range cmdDescriptor.commandApplier {
			applier(cmdIndex, cmdDescriptor.command)
		}

		//here we can free the applier (we don't need them anymore)
		//and such functions have the potential to "lock" some memory
		cmdDescriptor.commandApplier = nil

		err := cmdDescriptor.command.Start()
		if err != nil {
			return fmt.Errorf("failed to start command: %w", err)
		}
	}

	runErrors := runErrors()
	runErrors.errors = make([]error, len(c.cmdDescriptors))

	// here we have to wait in reversed order because if the last command will not read their stdin anymore
	// the previous command will wait endless for continuing writing to stdout
	for cmdIndex := len(c.cmdDescriptors) - 1; cmdIndex >= 0; cmdIndex-- {
		cmdDescriptor := c.cmdDescriptors[cmdIndex]

		err := cmdDescriptor.command.Wait()
		if closer, isCloser := cmdDescriptor.command.Stdin.(io.Closer); isCloser {
			// This is little hard to understand. Let's assume we have the chain: cmd1->cmd2
			//
			// For pipelining the commands together we will use the "StdoutPipe()"-Method of the cmd1. The result of
			// this method will be used as the Input-Stream of cmd2. But this pipe (cmd1.stdout -> cmd2.stdin) will be
			// closed normally only after cmd1 will be exited. And cmd1 will only exit after their job is done! But if
			// cmd2 will exit earlier (this can be happen if cmd2 will not consume the complete stdin-stream), cmd1 will
			// wait for eternity! To avoid that, we have to close the cmd2' input-stream manually!

			_ = closer.Close() // dont care about closing error
		}

		if err == nil {
			runErrors.setError(cmdIndex, nil)
		} else {
			shouldAdd := true

			if cmdDescriptor.errorChecker != nil {
				// let the corresponding error check decide if the error is "relevant" or not
				shouldAdd = cmdDescriptor.errorChecker(cmdIndex, cmdDescriptor.command, err)
			} else if c.errorChecker != nil {
				// let the global error check decide if the error is "relevant" or not
				shouldAdd = c.errorChecker(cmdIndex, cmdDescriptor.command, err)
			}

			if shouldAdd {
				runErrors.setError(cmdIndex, err)
			} else {
				runErrors.setError(cmdIndex, nil)
			}
		}
	}

	//according to documentation of command's StdoutPipe()/StderrPipe() we have to wait for all stream reads are done
	//after that we can wait for the commands:
	//   "[...] It is thus incorrect to call Wait before all reads from the pipe have completed. [...]"
	c.streamRoutinesWg.Wait()

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
