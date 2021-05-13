package command_chain

import "io"

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
