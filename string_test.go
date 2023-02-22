package cmdchain

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestChain_String(t *testing.T) {
	tests := []struct {
		c FinalizedBuilder
		e string
	}{
		{
			c: Builder().Join("echo", "hello world").Finalize(),
			e: `
[SO]                             ╿
[CM] /usr/bin/echo "hello world" ╡
[SE]                             ╽
`,
		},
		{
			c: Builder().Join("echo", `hello "world"`).Finalize(),
			e: `
[SO]                                 ╿
[CM] /usr/bin/echo "hello \"world\"" ╡
[SE]                                 ╽
`,
		},
		{
			c: Builder().
				Join("echo", "hello world").
				Finalize().WithOutput(&bytes.Buffer{}),
			e: `
[OS]                             ╭ *bytes.Buffer
[SO]                             │
[CM] /usr/bin/echo "hello world" ╡
[SE]                             ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").
				Finalize().WithOutput(&bytes.Buffer{}, &bytes.Buffer{}),
			e: `
[OS]                             ╭ *bytes.Buffer, *bytes.Buffer
[SO]                             │
[CM] /usr/bin/echo "hello world" ╡
[SE]                             ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").
				Finalize().WithError(&bytes.Buffer{}),
			e: `
[SO]                             ╿
[CM] /usr/bin/echo "hello world" ╡
[SE]                             │
[ES]                             ╰ *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").
				Finalize().WithOutput(&bytes.Buffer{}).WithError(&bytes.Buffer{}),
			e: `
[OS]                             ╭ *bytes.Buffer
[SO]                             │
[CM] /usr/bin/echo "hello world" ╡
[SE]                             │
[ES]                             ╰ *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS] *bytes.Buffer ╮
[OS]               │
[SO]               │                             ╿
[CM]               ╰ /usr/bin/echo "hello world" ╡
[SE]                                             ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithInjections(&bytes.Buffer{}, &bytes.Buffer{}).
				Finalize(),
			e: `
[IS] *bytes.Buffer, *bytes.Buffer ╮
[OS]                              │
[SO]                              │                             ╿
[CM]                              ╰ /usr/bin/echo "hello world" ╡
[SE]                                                            ╽
			`,
		},
		{
			c: Builder().WithInput(&bytes.Buffer{}).
				Join("echo", "hello world").
				Finalize(),
			e: `
[IS] *bytes.Buffer ╮
[OS]               │
[SO]               │                             ╿
[CM]               ╰ /usr/bin/echo "hello world" ╡
[SE]                                             ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().
				Join("grep", "hello").
				Finalize(),
			e: `
[SO]                             ╿                       ╿
[CM] /usr/bin/echo "hello world" ╡ /usr/bin/grep "hello" ╡
[SE]                             ╽                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").
				Finalize(),
			e: `
[SO]                             ╿                       ╿
[CM] /usr/bin/echo "hello world" ╡ /usr/bin/grep "hello" ╡
[SE]                             │                       ╽
[ES]                             ╰ *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().ForwardError().
				Join("grep", "hello").
				Finalize(),
			e: `
[SO]                             ╿                        ╿
[CM] /usr/bin/echo "hello world" ╡╭ /usr/bin/grep "hello" ╡
[SE]                             ╰╯                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().ForwardError().WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").
				Finalize(),
			e: `
[SO]                             ╿                        ╿
[CM] /usr/bin/echo "hello world" ╡╭ /usr/bin/grep "hello" ╡
[SE]                             ├╯                       ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").
				Join("grep", "hello").
				Finalize(),
			e: `
[SO]                             ╭╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╰ /usr/bin/grep "hello" ╡
[SE]                             ╽                        ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").
				Finalize(),
			e: `
[SO]                             ╭╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╰ /usr/bin/grep "hello" ╡
[SE]                             │                        ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").ForwardError().
				Join("grep", "hello").
				Finalize(),
			e: `
[SO]                             ╭╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╞ /usr/bin/grep "hello" ╡
[SE]                             ╰╯                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").ForwardError().WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").
				Finalize(),
			e: `
[SO]                             ╭╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╞ /usr/bin/grep "hello" ╡
[SE]                             ├╯                       ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().WithOutputForks(&bytes.Buffer{}).
				Join("grep", "hello").
				Finalize(),
			e: `
[OS]                             ╭ *bytes.Buffer
[SO]                             │                       ╿
[CM] /usr/bin/echo "hello world" ╡ /usr/bin/grep "hello" ╡
[SE]                             ╽                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().WithOutputForks(&bytes.Buffer{}).WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").
				Finalize(),
			e: `
[OS]                             ╭ *bytes.Buffer
[SO]                             │                       ╿
[CM] /usr/bin/echo "hello world" ╡ /usr/bin/grep "hello" ╡
[SE]                             │                       ╽
[ES]                             ╰ *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).DiscardStdOut().ForwardError().
				Join("grep", "hello").
				Finalize(),
			e: `
[OS]                             ╭  *bytes.Buffer
[SO]                             │                        ╿
[CM] /usr/bin/echo "hello world" ╡╭ /usr/bin/grep "hello" ╡
[SE]                             ╰╯                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).DiscardStdOut().ForwardError().WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").
				Finalize(),
			e: `
[OS]                             ╭  *bytes.Buffer
[SO]                             │                        ╿
[CM] /usr/bin/echo "hello world" ╡╭ /usr/bin/grep "hello" ╡
[SE]                             ├╯                       ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).
				Join("grep", "hello").
				Finalize(),
			e: `
[OS]                             ╭  *bytes.Buffer
[SO]                             ├╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╰ /usr/bin/grep "hello" ╡
[SE]                             ╽                        ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").
				Finalize(),
			e: `
[OS]                             ╭  *bytes.Buffer
[SO]                             ├╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╰ /usr/bin/grep "hello" ╡
[SE]                             │                        ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).ForwardError().
				Join("grep", "hello").
				Finalize(),
			e: `
[OS]                             ╭  *bytes.Buffer
[SO]                             ├╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╞ /usr/bin/grep "hello" ╡
[SE]                             ╰╯                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).ForwardError().WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").
				Finalize(),
			e: `
[OS]                             ╭  *bytes.Buffer
[SO]                             ├╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╞ /usr/bin/grep "hello" ╡
[SE]                             ├╯                       ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer  ╮
[OS]                              │
[SO]                             ╿│                       ╿
[CM] /usr/bin/echo "hello world" ╡╰ /usr/bin/grep "hello" ╡
[SE]                             ╽                        ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer  ╮
[OS]                              │
[SO]                             ╿│                       ╿
[CM] /usr/bin/echo "hello world" ╡╰ /usr/bin/grep "hello" ╡
[SE]                             │                        ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().ForwardError().
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer  ╮
[OS]                              │
[SO]                             ╿│                       ╿
[CM] /usr/bin/echo "hello world" ╡╞ /usr/bin/grep "hello" ╡
[SE]                             ╰╯                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().ForwardError().
				Join("grep", "hello").WithInjections(&bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}).
				Finalize(),
			e: `
[IS] *bytes.Buffer, *bytes.Buffer, *bytes.Buffer  ╮
[OS]                                              │
[SO]                                             ╿│                       ╿
[CM]                 /usr/bin/echo "hello world" ╡╞ /usr/bin/grep "hello" ╡
[SE]                                             ╰╯                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().ForwardError().WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer  ╮
[OS]                              │
[SO]                             ╿│                       ╿
[CM] /usr/bin/echo "hello world" ╡╞ /usr/bin/grep "hello" ╡
[SE]                             ├╯                       ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │
[SO]                             ├╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╰ /usr/bin/grep "hello" ╡
[SE]                             ╽                        ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │
[SO]                             ├╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╰ /usr/bin/grep "hello" ╡
[SE]                             │                        ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").ForwardError().
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │
[SO]                             ├╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╞ /usr/bin/grep "hello" ╡
[SE]                             ╰╯                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").ForwardError().WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │
[SO]                             ├╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╞ /usr/bin/grep "hello" ╡
[SE]                             ├╯                       ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().WithOutputForks(&bytes.Buffer{}).
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │╭─ *bytes.Buffer
[SO]                             ╰┿╮                       ╿
[CM] /usr/bin/echo "hello world" ═╡╰ /usr/bin/grep "hello" ╡
[SE]                              ╽                        ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().WithOutputForks(&bytes.Buffer{}).WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │╭─ *bytes.Buffer
[SO]                             ╰┿╮                       ╿
[CM] /usr/bin/echo "hello world" ═╡╰ /usr/bin/grep "hello" ╡
[SE]                              │                        ╽
[ES]                              ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").DiscardStdOut().WithOutputForks(&bytes.Buffer{}).ForwardError().
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │╭─ *bytes.Buffer
[SO]                             ╰┿╮                       ╿
[CM] /usr/bin/echo "hello world" ═╡╞ /usr/bin/grep "hello" ╡
[SE]                              ╰╯                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).DiscardStdOut().ForwardError().WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │╭─ *bytes.Buffer
[SO]                             ╰┿╮                       ╿
[CM] /usr/bin/echo "hello world" ═╡╞ /usr/bin/grep "hello" ╡
[SE]                              ├╯                       ╽
[ES]                              ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │╭─ *bytes.Buffer
[SO]                             ╰┼╮                       ╿
[CM] /usr/bin/echo "hello world" ═╡╰ /usr/bin/grep "hello" ╡
[SE]                              ╽                        ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │╭─ *bytes.Buffer
[SO]                             ╰┼╮                       ╿
[CM] /usr/bin/echo "hello world" ═╡╰ /usr/bin/grep "hello" ╡
[SE]                              │                        ╽
[ES]                              ╰  *bytes.Buffer
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).ForwardError().
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             │╭─ *bytes.Buffer
[SO]                             ╰┼╮                       ╿
[CM] /usr/bin/echo "hello world" ═╡╞ /usr/bin/grep "hello" ╡
[SE]                              ╰╯                       ╽
			`,
		},
		{
			c: Builder().
				Join("echo", "hello world").WithOutputForks(&bytes.Buffer{}).ForwardError().WithErrorForks(&bytes.Buffer{}).
				Join("grep", "hello").WithInjections(&bytes.Buffer{}).
				Finalize(),
			e: `
[IS]               *bytes.Buffer ╮
[OS]                             ├─ *bytes.Buffer
[SO]                             ├╮                       ╿
[CM] /usr/bin/echo "hello world" ╡╞ /usr/bin/grep "hello" ╡
[SE]                             ├╯                       ╽
[ES]                             ╰  *bytes.Buffer
			`,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("TestChain_String_%d", i), func(t *testing.T) {
			expected := ""

			for _, line := range strings.Split(tt.e, "\n") {
				if len(strings.TrimSpace(line)) == 0 {
					continue
				}

				if expected != "" {
					expected += "\n"
				}
				expected += line
			}

			given := tt.c.String()
			assert.Equal(t, expected, given)
		})
	}
}
