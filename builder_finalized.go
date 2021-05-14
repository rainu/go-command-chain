package cmdchain

import (
	"fmt"
	"io"
)

func (c *chain) WithOutput(targets ...io.Writer) FinalizedBuilder {
	if len(targets) == 1 {
		c.cmdDescriptors[len(c.cmdDescriptors)-1].command.Stdout = targets[0]
	} else if len(targets) > 1 {
		c.cmdDescriptors[len(c.cmdDescriptors)-1].command.Stdout = io.MultiWriter(targets...)
	}

	return c
}

func (c *chain) WithError(targets ...io.Writer) FinalizedBuilder {
	if len(targets) == 1 {
		c.cmdDescriptors[len(c.cmdDescriptors)-1].command.Stderr = targets[0]
	} else if len(targets) > 1 {
		c.cmdDescriptors[len(c.cmdDescriptors)-1].command.Stderr = io.MultiWriter(targets...)
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

	//according to documentation of command's StdoutPipe()/StderrPipe() we have to wait for all stream reads are done
	//after that we can wait for the commands:
	//   "[...] It is thus incorrect to call Wait before all reads from the pipe have completed. [...]"
	c.streamRoutinesWg.Wait()

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
