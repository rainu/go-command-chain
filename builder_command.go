package cmdchain

import "io"

func (c *chain) ForwardError() CommandBuilder {
	c.cmdDescriptors[len(c.cmdDescriptors)-1].errToIn = true
	return c
}

func (c *chain) DiscardStdOut() CommandBuilder {
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

func (c *chain) WithInjections(sources ...io.Reader) CommandBuilder {
	cmdDesc := c.cmdDescriptors[len(c.cmdDescriptors)-1]

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

		if len(combineSrc) > 0 {
			var err error
			cmdDesc.command.Stdin, err = c.combineStream(combineSrc...)
			if err != nil {
				c.streamErrors.setError(len(c.cmdDescriptors)-1, err)
			}
		}
	}

	return c
}
