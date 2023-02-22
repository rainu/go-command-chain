package cmdchain

import (
	"fmt"
	"io"
	"os"
)

func (c *chain) Apply(applier CommandApplier) CommandBuilder {
	applier(len(c.cmdDescriptors)-1, c.cmdDescriptors[len(c.cmdDescriptors)-1].command)
	return c
}

func (c *chain) ApplyBeforeStart(applier CommandApplier) CommandBuilder {
	i := len(c.cmdDescriptors) - 1
	c.cmdDescriptors[i].commandApplier = append(c.cmdDescriptors[i].commandApplier, applier)

	return c
}

func (c *chain) ForwardError() CommandBuilder {
	c.cmdDescriptors[len(c.cmdDescriptors)-1].errToIn = true
	return c
}

func (c *chain) DiscardStdOut() CommandBuilder {
	c.cmdDescriptors[len(c.cmdDescriptors)-1].outToIn = false
	return c
}

func (c *chain) WithOutputForks(targets ...io.Writer) CommandBuilder {
	cmdDesc := &(c.cmdDescriptors[len(c.cmdDescriptors)-1])
	cmdDesc.outputStreams = append(cmdDesc.outputStreams, targets...)

	if len(targets) > 1 {
		cmdDesc.outFork = io.MultiWriter(targets...)
	} else if len(targets) == 1 {
		cmdDesc.outFork = targets[0]
	}

	return c
}

func (c *chain) WithErrorForks(targets ...io.Writer) CommandBuilder {
	cmdDesc := &(c.cmdDescriptors[len(c.cmdDescriptors)-1])
	cmdDesc.errorStreams = append(cmdDesc.errorStreams, targets...)

	if len(targets) > 1 {
		cmdDesc.errFork = io.MultiWriter(targets...)
	} else if len(targets) == 1 {
		cmdDesc.errFork = targets[0]
	}
	return c
}

func (c *chain) WithInjections(sources ...io.Reader) CommandBuilder {
	cmdDesc := &(c.cmdDescriptors[len(c.cmdDescriptors)-1])
	cmdDesc.inputStreams = append(cmdDesc.inputStreams, sources...)

	if len(sources) > 0 {
		combineSrc := make([]io.Reader, 0, len(sources)+1)
		if cmdDesc.command.Stdin != nil {
			combineSrc = append(combineSrc, cmdDesc.command.Stdin)
		}

		for _, source := range sources {
			if source != nil {
				combineSrc = append(combineSrc, source)
			}
		}

		if len(combineSrc) == 1 {
			cmdDesc.command.Stdin = combineSrc[0]
		} else if len(combineSrc) > 1 {
			var err error
			cmdDesc.command.Stdin, err = c.combineStream(combineSrc...)
			if err != nil {
				c.streamErrors.setError(len(c.cmdDescriptors)-1, err)
			}
		}
	}

	return c
}

func (c *chain) WithEnvironmentMap(envMap map[interface{}]interface{}) CommandBuilder {
	cmdDesc := c.cmdDescriptors[len(c.cmdDescriptors)-1]

	for key, value := range envMap {
		cmdDesc.command.Env = append(cmdDesc.command.Env, fmt.Sprintf("%v=%v", key, value))
	}
	return c
}

func (c *chain) WithEnvironment(envMap ...interface{}) CommandBuilder {
	if len(envMap)%2 != 0 {
		c.buildErrors.addError(fmt.Errorf("invalid count of environment arguments"))
		return c
	}
	cmdDesc := c.cmdDescriptors[len(c.cmdDescriptors)-1]

	for i := 0; i < len(envMap); i += 2 {
		cmdDesc.command.Env = append(cmdDesc.command.Env, fmt.Sprintf("%v=%v", envMap[i], envMap[i+1]))
	}
	return c
}

func (c *chain) WithAdditionalEnvironmentMap(envMap map[interface{}]interface{}) CommandBuilder {
	cmdDesc := c.cmdDescriptors[len(c.cmdDescriptors)-1]
	if len(cmdDesc.command.Env) == 0 {
		cmdDesc.command.Env = os.Environ()
	}

	return c.WithEnvironmentMap(envMap)
}

func (c *chain) WithAdditionalEnvironment(envMap ...interface{}) CommandBuilder {
	cmdDesc := c.cmdDescriptors[len(c.cmdDescriptors)-1]
	if len(cmdDesc.command.Env) == 0 {
		cmdDesc.command.Env = os.Environ()
	}

	return c.WithEnvironment(envMap...)
}

func (c *chain) WithWorkingDirectory(workingDir string) CommandBuilder {
	cmdDesc := c.cmdDescriptors[len(c.cmdDescriptors)-1]
	cmdDesc.command.Dir = workingDir
	return c
}

func (c *chain) WithErrorChecker(errChecker ErrorChecker) CommandBuilder {
	c.cmdDescriptors[len(c.cmdDescriptors)-1].errorChecker = errChecker
	return c
}
