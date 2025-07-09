package cmdchain

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func (c *cmdDescriptor) String() string {
	out := strings.Builder{}

	out.WriteString(c.command.Path)
	for _, arg := range c.command.Args[1:] {
		out.WriteString(" " + strconv.Quote(arg))
	}

	return out.String()
}

type stringModel struct {
	Chunks []modelChunk
}

type modelChunk struct {
	InputStream  string
	OutputStream string
	Command      string
	ErrorStream  string

	Pipe pipe
}

func (c *modelChunk) Space() (space int) {
	space = len(c.InputStream)

	if len(c.OutputStream) > space {
		space = len(c.OutputStream)
	}
	if len(c.Command) > space {
		space = len(c.Command)
	}
	if len(c.ErrorStream) > space {
		space = len(c.ErrorStream)
	}

	return space
}

type pipe [6]string
type pipeVariation struct {
	pipe       pipe
	isValidFor func(current, next *cmdDescriptor) bool
}

func (c *cmdDescriptor) hasInputStreams() bool  { return len(c.inputStreams) > 0 }
func (c *cmdDescriptor) hasOutputStreams() bool { return len(c.outputStreams) > 0 }
func (c *cmdDescriptor) hasErrorStreams() bool  { return len(c.errorStreams) > 0 }

var availablePipes = []pipeVariation{
	{
		pipe{
			"   ",
			"   ",
			" ╿ ",
			" ╡ ",
			" ╽ ",
			"   ",
		},
		func(c, n *cmdDescriptor) bool {
			// is last command
			if n != nil {
				return false
			}

			return !c.hasOutputStreams() && !c.hasErrorStreams()
		},
	},
	{
		pipe{
			"   ",
			" ╭ ",
			" │ ",
			" ╡ ",
			" ╽ ",
			"   ",
		},
		func(c, n *cmdDescriptor) bool {
			// is last command
			if n != nil {
				return false
			}

			return c.hasOutputStreams() && !c.hasInputStreams() && !c.hasErrorStreams()
		},
	},
	{
		pipe{
			"   ",
			"   ",
			" ╿ ",
			" ╡ ",
			" │ ",
			" ╰ ",
		},
		func(c, n *cmdDescriptor) bool {
			// is last command
			if n != nil {
				return false
			}

			return c.hasErrorStreams() && !c.hasInputStreams() && !c.hasOutputStreams()
		},
	},
	{
		pipe{
			"   ",
			" ╭ ",
			" │ ",
			" ╡ ",
			" │ ",
			" ╰ ",
		},
		func(c, n *cmdDescriptor) bool {
			// is last command
			if n != nil {
				return false
			}

			return c.hasErrorStreams() && c.hasOutputStreams() && !c.hasInputStreams()
		},
	},
	{
		pipe{
			" ╮ ",
			" │ ",
			" │ ",
			" ╰ ",
			"   ",
			"   ",
		},
		func(c, n *cmdDescriptor) bool {
			// is first command
			if c != nil {
				return false
			}

			return n.hasInputStreams()
		},
	},
	{
		pipe{
			"   ",
			"   ",
			" ╿ ",
			" ╡ ",
			" ╽ ",
			"   ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && !c.hasOutputStreams() && !c.outToIn && !c.errToIn && !c.hasErrorStreams() //0
		},
	},
	{
		pipe{
			"   ",
			"   ",
			" ╿ ",
			" ╡ ",
			" │ ",
			" ╰ ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && !c.hasOutputStreams() && !c.outToIn && !c.errToIn && c.hasErrorStreams() //1
		},
	},
	{
		pipe{
			"    ",
			"    ",
			" ╿  ",
			" ╡╭ ",
			" ╰╯ ",
			"    ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && !c.hasOutputStreams() && !c.outToIn && c.errToIn && !c.hasErrorStreams() //2
		},
	},
	{
		pipe{
			"    ",
			"    ",
			" ╿  ",
			" ╡╭ ",
			" ├╯ ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && !c.hasOutputStreams() && !c.outToIn && c.errToIn && c.hasErrorStreams() //3
		},
	},
	{
		pipe{
			"    ",
			"    ",
			" ╭╮ ",
			" ╡╰ ",
			" ╽  ",
			"    ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && !c.hasOutputStreams() && c.outToIn && !c.errToIn && !c.hasErrorStreams() //4
		},
	},
	{
		pipe{
			"    ",
			"    ",
			" ╭╮ ",
			" ╡╰ ",
			" │  ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && !c.hasOutputStreams() && c.outToIn && !c.errToIn && c.hasErrorStreams() //5
		},
	},
	{
		pipe{
			"    ",
			"    ",
			" ╭╮ ",
			" ╡╞ ",
			" ╰╯ ",
			"    ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && !c.hasOutputStreams() && c.outToIn && c.errToIn && !c.hasErrorStreams() //6
		},
	},
	{
		pipe{
			"    ",
			"    ",
			" ╭╮ ",
			" ╡╞ ",
			" ├╯ ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && !c.hasOutputStreams() && c.outToIn && c.errToIn && c.hasErrorStreams() //7
		},
	},
	{
		pipe{
			"   ",
			" ╭ ",
			" │ ",
			" ╡ ",
			" ╽ ",
			"   ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && c.hasOutputStreams() && !c.outToIn && !c.errToIn && !c.hasErrorStreams() //8
		},
	},
	{
		pipe{
			"   ",
			" ╭ ",
			" │ ",
			" ╡ ",
			" │ ",
			" ╰ ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && c.hasOutputStreams() && !c.outToIn && !c.errToIn && c.hasErrorStreams() //9
		},
	},
	{
		pipe{
			"    ",
			" ╭  ",
			" │  ",
			" ╡╭ ",
			" ╰╯ ",
			"    ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && c.hasOutputStreams() && !c.outToIn && c.errToIn && !c.hasErrorStreams() //10
		},
	},
	{
		pipe{
			"    ",
			" ╭  ",
			" │  ",
			" ╡╭ ",
			" ├╯ ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && c.hasOutputStreams() && !c.outToIn && c.errToIn && c.hasErrorStreams() //11
		},
	},
	{
		pipe{
			"    ",
			" ╭  ",
			" ├╮ ",
			" ╡╰ ",
			" ╽  ",
			"    ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && c.hasOutputStreams() && c.outToIn && !c.errToIn && !c.hasErrorStreams() //12
		},
	},
	{
		pipe{
			"    ",
			" ╭  ",
			" ├╮ ",
			" ╡╰ ",
			" │  ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && c.hasOutputStreams() && c.outToIn && !c.errToIn && c.hasErrorStreams() //13
		},
	},
	{
		pipe{
			"    ",
			" ╭  ",
			" ├╮ ",
			" ╡╞ ",
			" ╰╯ ",
			"    ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && c.hasOutputStreams() && c.outToIn && c.errToIn && !c.hasErrorStreams() //14
		},
	},
	{
		pipe{
			"    ",
			" ╭  ",
			" ├╮ ",
			" ╡╞ ",
			" ├╯ ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return !n.hasInputStreams() && c.hasOutputStreams() && c.outToIn && c.errToIn && c.hasErrorStreams() //15
		},
	},
	{
		pipe{
			"  ╮ ",
			"  │ ",
			" ╿│ ",
			" ╡╰ ",
			" ╽  ",
			"    ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && !c.hasOutputStreams() && !c.outToIn && !c.errToIn && !c.hasErrorStreams() //16
		},
	},
	{
		pipe{
			"  ╮ ",
			"  │ ",
			" ╿│ ",
			" ╡╰ ",
			" │  ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && !c.hasOutputStreams() && !c.outToIn && !c.errToIn && c.hasErrorStreams() //17
		},
	},
	{
		pipe{
			"  ╮ ",
			"  │ ",
			" ╿│ ",
			" ╡╞ ",
			" ╰╯ ",
			"    ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && !c.hasOutputStreams() && !c.outToIn && c.errToIn && !c.hasErrorStreams() //18
		},
	},
	{
		pipe{
			"  ╮ ",
			"  │ ",
			" ╿│ ",
			" ╡╞ ",
			" ├╯ ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && !c.hasOutputStreams() && !c.outToIn && c.errToIn && c.hasErrorStreams() //19
		},
	},
	{
		pipe{
			" ╮  ",
			" │  ",
			" ├╮ ",
			" ╡╰ ",
			" ╽  ",
			"    ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && !c.hasOutputStreams() && c.outToIn && !c.errToIn && !c.hasErrorStreams() //20
		},
	},
	{
		pipe{
			" ╮  ",
			" │  ",
			" ├╮ ",
			" ╡╰ ",
			" │  ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && !c.hasOutputStreams() && c.outToIn && !c.errToIn && c.hasErrorStreams() //21
		},
	},
	{
		pipe{
			" ╮  ",
			" │  ",
			" ├╮ ",
			" ╡╞ ",
			" ╰╯ ",
			"    ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && !c.hasOutputStreams() && c.outToIn && c.errToIn && !c.hasErrorStreams() //22
		},
	},
	{
		pipe{
			" ╮  ",
			" │  ",
			" ├╮ ",
			" ╡╞ ",
			" ├╯ ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && !c.hasOutputStreams() && c.outToIn && c.errToIn && c.hasErrorStreams() //23
		},
	},
	{
		pipe{
			" ╮   ",
			" │╭─ ",
			" ╰┿╮ ",
			" ═╡╰ ",
			"  ╽  ",
			"     ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && c.hasOutputStreams() && !c.outToIn && !c.errToIn && !c.hasErrorStreams() //24
		},
	},
	{
		pipe{
			" ╮   ",
			" │╭─ ",
			" ╰┿╮ ",
			" ═╡╰ ",
			"  │  ",
			"  ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && c.hasOutputStreams() && !c.outToIn && !c.errToIn && c.hasErrorStreams() //25
		},
	},
	{
		pipe{
			" ╮   ",
			" │╭─ ",
			" ╰┿╮ ",
			" ═╡╞ ",
			"  ╰╯ ",
			"     ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && c.hasOutputStreams() && !c.outToIn && c.errToIn && !c.hasErrorStreams() //26
		},
	},
	{
		pipe{
			" ╮   ",
			" │╭─ ",
			" ╰┿╮ ",
			" ═╡╞ ",
			"  ├╯ ",
			"  ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && c.hasOutputStreams() && !c.outToIn && c.errToIn && c.hasErrorStreams() //27
		},
	},
	{
		pipe{
			" ╮   ",
			" │╭─ ",
			" ╰┼╮ ",
			" ═╡╰ ",
			"  ╽  ",
			"     ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && c.hasOutputStreams() && c.outToIn && !c.errToIn && !c.hasErrorStreams() //28
		},
	},
	{
		pipe{
			" ╮   ",
			" │╭─ ",
			" ╰┼╮ ",
			" ═╡╰ ",
			"  │  ",
			"  ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && c.hasOutputStreams() && c.outToIn && !c.errToIn && c.hasErrorStreams() //29
		},
	},
	{
		pipe{
			" ╮   ",
			" │╭─ ",
			" ╰┼╮ ",
			" ═╡╞ ",
			"  ╰╯ ",
			"     ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && c.hasOutputStreams() && c.outToIn && c.errToIn && !c.hasErrorStreams() //30
		},
	},
	{
		pipe{
			" ╮  ",
			" ├─ ",
			" ├╮ ",
			" ╡╞ ",
			" ├╯ ",
			" ╰  ",
		},
		func(c, n *cmdDescriptor) bool {
			if c == nil || n == nil {
				return false
			}

			return n.hasInputStreams() && c.hasOutputStreams() && c.outToIn && c.errToIn && c.hasErrorStreams() //31
		},
	},
	{
		pipe{"", "", "", "", "", ""},
		func(c, n *cmdDescriptor) bool { return true },
	},
}

func findPipe(c, n *cmdDescriptor) pipe {
	for _, p := range availablePipes {
		if p.isValidFor(c, n) {
			return p.pipe
		}
	}

	//should never happen
	return pipe{}
}

func (c *chain) toStringModel() stringModel {
	model := stringModel{
		Chunks: make([]modelChunk, len(c.cmdDescriptors)+2, len(c.cmdDescriptors)+2),
	}
	model.Chunks[0].Pipe = findPipe(nil, &c.cmdDescriptors[0])

	for i, cmdDesc := range c.cmdDescriptors {
		i++

		//isFirst := i == 1
		isLast := i == len(c.cmdDescriptors)
		prevChunk := &model.Chunks[i-1]
		curChunk := &model.Chunks[i]
		nextChunk := &model.Chunks[i+1]

		////
		// input stream line
		////
		if len(cmdDesc.inputStreams) > 0 {
			streamTypes := make([]string, len(cmdDesc.inputStreams), len(cmdDesc.inputStreams))
			for j, inputStream := range cmdDesc.inputStreams {
				streamTypes[j] = streamString(inputStream)
			}
			prevChunk.InputStream = strings.Join(streamTypes, ", ")
		}

		////
		// output stream line
		////
		if len(cmdDesc.outputStreams) > 0 {
			streamTypes := make([]string, len(cmdDesc.outputStreams), len(cmdDesc.outputStreams))
			for j, outputStream := range cmdDesc.outputStreams {
				streamTypes[j] = streamString(outputStream)
			}
			nextChunk.OutputStream = strings.Join(streamTypes, ", ")

		}

		////
		// command line
		////
		curChunk.Command = cmdDesc.String()

		////
		// error stream line
		////
		if len(cmdDesc.errorStreams) > 0 {
			streamTypes := make([]string, len(cmdDesc.errorStreams), len(cmdDesc.errorStreams))
			for j, errorStream := range cmdDesc.errorStreams {
				streamTypes[j] = streamString(errorStream)
			}
			nextChunk.ErrorStream = strings.Join(streamTypes, ", ")
		}

		////
		// pipes
		////
		if !isLast {
			curChunk.Pipe = findPipe(&cmdDesc, &c.cmdDescriptors[i])
		} else {
			curChunk.Pipe = findPipe(&cmdDesc, nil)
		}
	}

	return model
}

func streamString(stream any) (s string) {
	if stringer, ok := stream.(fmt.Stringer); ok {
		s = stringer.String()
	}
	if len(s) == 0 {
		if file, ok := stream.(*os.File); ok {
			s = file.Name()
		}
	}
	if len(s) == 0 {
		s = fmt.Sprintf("%s", reflect.TypeOf(stream))
	}

	return
}

func (s *stringModel) String() string {
	inStreamLane := &strings.Builder{}
	outStreamLane := &strings.Builder{}
	outLane := &strings.Builder{}
	cmdLane := &strings.Builder{}
	errLane := &strings.Builder{}
	errStreamLane := &strings.Builder{}

	// we should have at least three chunks
	if len(s.Chunks) < 3 {
		return ""
	}

	for _, chunk := range s.Chunks {
		chunkSpace := chunk.Space()

		inStreamLane.WriteString(strings.Repeat(" ", chunkSpace-len(chunk.InputStream)))
		inStreamLane.WriteString(chunk.InputStream)
		inStreamLane.WriteString(chunk.Pipe[0])

		outStreamLane.WriteString(chunk.OutputStream)
		outStreamLane.WriteString(strings.Repeat(" ", chunkSpace-len(chunk.OutputStream)))
		outStreamLane.WriteString(chunk.Pipe[1])

		outLane.WriteString(strings.Repeat(" ", chunkSpace))
		outLane.WriteString(chunk.Pipe[2])

		cmdLane.WriteString(strings.Repeat(" ", chunkSpace-len(chunk.Command)))
		cmdLane.WriteString(chunk.Command)
		cmdLane.WriteString(chunk.Pipe[3])

		errLane.WriteString(strings.Repeat(" ", chunkSpace))
		errLane.WriteString(chunk.Pipe[4])

		errStreamLane.WriteString(chunk.ErrorStream)
		errStreamLane.WriteString(strings.Repeat(" ", chunkSpace-len(chunk.ErrorStream)))
		errStreamLane.WriteString(chunk.Pipe[5])
	}

	result := ""

	if len(strings.TrimSpace(inStreamLane.String())) > 0 {
		result += "[IS] "
		result += strings.TrimRight(inStreamLane.String(), " ") + "\n"
	}
	if len(strings.TrimSpace(outStreamLane.String())) > 0 {
		result += "[OS] "
		result += strings.TrimRight(outStreamLane.String(), " ") + "\n"
	}
	if len(strings.TrimSpace(outLane.String())) > 0 {
		result += "[SO] "
		result += strings.TrimRight(outLane.String(), " ") + "\n"
	}
	result += "[CM] "
	result += strings.TrimRight(cmdLane.String(), " ")
	if len(strings.TrimSpace(errLane.String())) > 0 {
		result += "\n[SE] "
		result += strings.TrimRight(errLane.String(), " ")
	}
	if len(strings.TrimSpace(errStreamLane.String())) > 0 {
		result += "\n[ES] "
		result += strings.TrimRight(errStreamLane.String(), " ")
	}

	return result
}

func (c *chain) String() string {
	model := c.toStringModel()
	return model.String()
}
