package cmdchain

type hook interface {
	BeforeRun()
	AfterRun()
}

func (c *chain) addHook(h hook) {
	c.hooks = append(c.hooks, h)
}

func (c *chain) executeBeforeRunHooks() {
	for _, h := range c.hooks {
		h.BeforeRun()
	}
}

func (c *chain) executeAfterRunHooks() {
	for _, h := range c.hooks {
		h.AfterRun()
	}
}
