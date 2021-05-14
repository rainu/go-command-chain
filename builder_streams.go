package cmdchain

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
)

func (c *chain) linkStreams(cmd *exec.Cmd) {
	//link this command's input with the previous command's output (cmd1 -> cmd2)
	prevCmdDesc := c.cmdDescriptors[len(c.cmdDescriptors)-2]

	var prevOut, prevErr io.ReadCloser
	var err error

	defer func() {
		c.buildErrors.addError(err)
	}()

	if prevCmdDesc.outToIn {
		prevOut, err = prevCmdDesc.command.StdoutPipe()
		if err != nil {
			return
		}
	} else if prevCmdDesc.outFork != nil {
		prevCmdDesc.command.Stdout = prevCmdDesc.outFork
	}

	if prevCmdDesc.errToIn {
		prevErr, err = prevCmdDesc.command.StderrPipe()
		if err != nil {
			return
		}
	} else if prevCmdDesc.errFork != nil {
		prevCmdDesc.command.Stderr = prevCmdDesc.errFork
	}

	if prevCmdDesc.outToIn && !prevCmdDesc.errToIn {
		if prevCmdDesc.outFork == nil {
			cmd.Stdin = prevOut
		} else {
			cmd.Stdin, err = c.forkStream(prevOut, prevCmdDesc.outFork)
		}
	} else if !prevCmdDesc.outToIn && prevCmdDesc.errToIn {
		if prevCmdDesc.errFork == nil {
			cmd.Stdin = prevErr
		} else {
			cmd.Stdin, err = c.forkStream(prevErr, prevCmdDesc.errFork)
		}
	} else if prevCmdDesc.outToIn && prevCmdDesc.errToIn {
		var outR io.Reader = prevOut
		var errR io.Reader = prevErr

		if prevCmdDesc.outFork != nil {
			outR, err = c.forkStream(prevOut, prevCmdDesc.outFork)
			if err != nil {
				return
			}
		}
		if prevCmdDesc.errFork != nil {
			errR, err = c.forkStream(prevErr, prevCmdDesc.errFork)
			if err != nil {
				return
			}
		}

		cmd.Stdin, err = c.combineStream(outR, errR)
	} else {
		//this should never be happen!
		err = errors.New("invalid stream configuration")
	}
}

func (c *chain) forkStream(src io.ReadCloser, target io.Writer) (io.Reader, error) {
	//initialise pipe and copy content inside own goroutine
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	/*
		+------+          +------+
		| cmd1 | ---+---> | cmd2 |
		+------+    |     +------+
					V
				 +---------+
				 | outFork |
				 +---------+
	*/

	c.streamRoutinesWg.Add(1)
	go func(cmdIndex int, src io.Reader) {
		//we have to make sure, the pipe will be closed after the prevCommand
		//have closed their output stream - otherwise this will cause a never
		//ending wait for finishing the command execution!
		defer pipeWriter.Close()
		defer c.streamRoutinesWg.Done()

		//the cmdOut must be written into both writer: outFork and pipeWriter.
		//input from pipeWriter will redirected to pipeReader (the input for
		//the next command)
		_, err := io.Copy(io.MultiWriter(pipeWriter, target), src)
		c.streamErrors.setError(cmdIndex, err)
	}(len(c.cmdDescriptors)-1, src)

	return pipeReader, nil
}

func (c *chain) combineStream(sources ...io.Reader) (*os.File, error) {
	cmdIndex := len(c.cmdDescriptors) - 1
	return c.combineStreamForCommand(cmdIndex, sources...)
}

func (c *chain) combineStreamForCommand(cmdIndex int, sources ...io.Reader) (*os.File, error) {
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	streamErrors := MultipleErrors{
		errors: make([]error, len(sources)),
	}

	wg := sync.WaitGroup{}
	wg.Add(len(sources))

	for i, src := range sources {

		//spawn goroutine for each stream to ensure the sources
		//will read in parallel
		go func(i int, src io.Reader) {
			defer wg.Done()

			_, err := io.Copy(pipeWriter, src)
			if err != nil {
				streamErrors.setError(i, err)
			}
		}(i, src)
	}

	c.streamRoutinesWg.Add(1)
	go func() {
		//we have to make sure that the pipe will be closed after all source streams
		//are read. otherwise this will cause a never ending wait for finishing the command execution!
		defer pipeWriter.Close()
		defer c.streamErrors.setError(cmdIndex, streamErrors)
		defer c.streamRoutinesWg.Done()

		//wait until all streams are read
		wg.Wait()
	}()

	return pipeReader, nil
}
