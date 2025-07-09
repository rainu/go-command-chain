package cmdchain

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestJoinShellCmd(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		expectedString string
		expectError    string
		check          func(*testing.T, *chain)
	}{
		{
			name:    "simple",
			command: `date`,
			expectedString: `
[SO]               ╿
[CM] /usr/bin/date ╡
[SE]               ╽
`,
		},
		{
			name:    "simple double quoted",
			command: `echo "Hello, World!"`,
			expectedString: `
[SO]                               ╿
[CM] /usr/bin/echo "Hello, World!" ╡
[SE]                               ╽
`,
		},
		{
			name:    "simple single quoted",
			command: `echo 'Hello, World!'`,
			expectedString: `
[SO]                               ╿
[CM] /usr/bin/echo "Hello, World!" ╡
[SE]                               ╽
`,
		},
		{
			name:    "simple non-quoted",
			command: `echo Hello, World!`,
			expectedString: `
[SO]                                 ╿
[CM] /usr/bin/echo "Hello," "World!" ╡
[SE]                                 ╽
`,
		},
		{
			name:    "simple chain double quoted",
			command: `echo "Hello, World!" | grep "Hello" | wc -c`,
			expectedString: `
[SO]                               ╭╮                       ╭╮                  ╿
[CM] /usr/bin/echo "Hello, World!" ╡╰ /usr/bin/grep "Hello" ╡╰ /usr/bin/wc "-c" ╡
[SE]                               ╽                        ╽                   ╽
`,
		},
		{
			name:    "simple chain single quoted",
			command: `echo 'Hello, World!' | grep 'Hello' | wc -c`,
			expectedString: `
[SO]                               ╭╮                       ╭╮                  ╿
[CM] /usr/bin/echo "Hello, World!" ╡╰ /usr/bin/grep "Hello" ╡╰ /usr/bin/wc "-c" ╡
[SE]                               ╽                        ╽                   ╽
`,
		},
		{
			name:    "simple chain non-quoted",
			command: `echo Hello, World! | grep 'Hello' | wc -c`,
			expectedString: `
[SO]                                 ╭╮                       ╭╮                  ╿
[CM] /usr/bin/echo "Hello," "World!" ╡╰ /usr/bin/grep "Hello" ╡╰ /usr/bin/wc "-c" ╡
[SE]                                 ╽                        ╽                   ╽
`,
		},
		{
			name:    "forward error chain",
			command: `echo Hello, World! |& grep 'Hello' |& wc -c`,
			expectedString: `
[SO]                                 ╭╮                       ╭╮                  ╿
[CM] /usr/bin/echo "Hello," "World!" ╡╞ /usr/bin/grep "Hello" ╡╞ /usr/bin/wc "-c" ╡
[SE]                                 ╰╯                       ╰╯                  ╽
`,
		},
		{
			name:    "local environment variable",
			command: `MY_VAR=1 date`,
			expectedString: `
[SO]               ╿
[CM] /usr/bin/date ╡
[SE]               ╽
`,
			check: func(t *testing.T, chain *chain) {
				assert.Contains(t, chain.cmdDescriptors[0].command.Env, "MY_VAR=1")
			},
		},
		{
			name:    "local environment variables",
			command: `MY_VAR=1 MY_SEC_VAR=2 date`,
			expectedString: `
[SO]               ╿
[CM] /usr/bin/date ╡
[SE]               ╽
`,
			check: func(t *testing.T, chain *chain) {
				assert.Contains(t, chain.cmdDescriptors[0].command.Env, "MY_VAR=1")
				assert.Contains(t, chain.cmdDescriptors[0].command.Env, "MY_SEC_VAR=2")
			},
		},
		{
			name:    "duplicate local environment variables",
			command: `MY_VAR=1 MY_VAR=2 date`,
			expectedString: `
[SO]               ╿
[CM] /usr/bin/date ╡
[SE]               ╽
`,
			check: func(t *testing.T, chain *chain) {
				assert.Contains(t, chain.cmdDescriptors[0].command.Env, "MY_VAR=1")
				assert.Contains(t, chain.cmdDescriptors[0].command.Env, "MY_VAR=2")
			},
		},
		{
			name:    "empty environment variable",
			command: `MY_VAR= date`,
			expectedString: `
[SO]               ╿
[CM] /usr/bin/date ╡
[SE]               ╽
`,
			check: func(t *testing.T, chain *chain) {
				assert.Contains(t, chain.cmdDescriptors[0].command.Env, "MY_VAR=")
			},
		},
		{
			name:        "no statements",
			command:     ``,
			expectError: "no statements",
		},
		{
			name:        "multiple statements",
			command:     `date; date`,
			expectError: "multiple statements are not supported, found 2 statements",
		},
		{
			name:        "logical OR concatenation",
			command:     `date || date`,
			expectError: "[1:1 - 1:13] unsupported binary operator '||' at '1:6'",
		},
		{
			name:        "logical AND concatenation",
			command:     `date && date`,
			expectError: "[1:1 - 1:13] unsupported binary operator '&&' at '1:6'",
		},
		{
			name:        "background execution",
			command:     `date &`,
			expectError: "background execution is not supported",
		},
		{
			name:    "file redirection",
			command: `date > /tmp/out |& grep 'Hello'`,
			expectedString: `
[OS]               ╭  *os.File
[SO]               ├╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ╰╯                       ╽
`,
			check: func(t *testing.T, c *chain) {
				assert.Equal(t, "/tmp/out", c.cmdDescriptors[0].outputStreams[0].(*os.File).Name())
				assert.Empty(t, c.cmdDescriptors[0].errorStreams)
			},
		},
		{
			name:    "file redirection (appending)",
			command: `date >> /tmp/out |& grep 'Hello'`,
			expectedString: `
[OS]               ╭  *os.File
[SO]               ├╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ╰╯                       ╽
`,
			check: func(t *testing.T, c *chain) {
				assert.Equal(t, "/tmp/out", c.cmdDescriptors[0].outputStreams[0].(*os.File).Name())
				assert.Empty(t, c.cmdDescriptors[0].errorStreams)
			},
		},
		{
			name:    "file redirection (error)",
			command: `date 2> /tmp/out |& grep 'Hello'`,
			expectedString: `
[OS]               ╭  *os.File
[SO]               ├╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ╰╯                       ╽
`,
			check: func(t *testing.T, c *chain) {
				assert.Equal(t, "/tmp/out", c.cmdDescriptors[0].outputStreams[0].(*os.File).Name())
				assert.Empty(t, c.cmdDescriptors[0].errorStreams)
			},
		},
		{
			name:    "file redirection (error appending)",
			command: `date 2>> /tmp/out |& grep 'Hello'`,
			expectedString: `
[OS]               ╭  *os.File
[SO]               ├╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ╰╯                       ╽
`,
			check: func(t *testing.T, c *chain) {
				assert.Equal(t, "/tmp/out", c.cmdDescriptors[0].outputStreams[0].(*os.File).Name())
				assert.Empty(t, c.cmdDescriptors[0].errorStreams)
			},
		},
		{
			name:    "both file redirection",
			command: `date &> /tmp/out |& grep 'Hello'`,
			expectedString: `
[OS]               ╭  *os.File
[SO]               ├╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ├╯                       ╽
[ES]               ╰  *os.File
`,
			check: func(t *testing.T, c *chain) {
				assert.Equal(t, "/tmp/out", c.cmdDescriptors[0].outputStreams[0].(*os.File).Name())
				assert.Equal(t, "/tmp/out", c.cmdDescriptors[0].errorStreams[0].(*os.File).Name())
			},
		},
		{
			name:    "both file redirection (appending)",
			command: `date &>> /tmp/out |& grep 'Hello'`,
			expectedString: `
[OS]               ╭  *os.File
[SO]               ├╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ├╯                       ╽
[ES]               ╰  *os.File
`,
			check: func(t *testing.T, c *chain) {
				assert.Equal(t, "/tmp/out", c.cmdDescriptors[0].outputStreams[0].(*os.File).Name())
				assert.Equal(t, "/tmp/out", c.cmdDescriptors[0].errorStreams[0].(*os.File).Name())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Builder().JoinShellCmd(tt.command).(*chain)

			var err error
			if result.buildErrors.hasError {
				err = result.buildErrors
			}

			if tt.expectError == "" {
				require.NoError(t, err)
				assert.Equal(t, strings.TrimSpace(tt.expectedString), strings.TrimSpace(result.String()))

				if tt.check != nil {
					tt.check(t, result)
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
			}
		})
	}
}

func TestJoinShellCmd_Multiple(t *testing.T) {
	c := Builder().
		JoinShellCmd("echo 'Hello, World!' | grep 'Hello'").
		JoinShellCmd("wc -l | grep '1'")

	expectedString := `
[SO]                               ╭╮                       ╭╮                  ╭╮                   ╿
[CM] /usr/bin/echo "Hello, World!" ╡╰ /usr/bin/grep "Hello" ╡╰ /usr/bin/wc "-l" ╡╰ /usr/bin/grep "1" ╡
[SE]                               ╽                        ╽                   ╽                    ╽
`
	assert.Equal(t, strings.TrimSpace(expectedString), strings.TrimSpace(c.Finalize().String()))
}

func TestJoinShellCmdWithContext(t *testing.T) {
	result := Builder().JoinShellCmdWithContext(t.Context(), `echo "hello world" | grep "hello" | wc -c`).(*chain)

	assert.False(t, result.buildErrors.hasError)
	assert.Len(t, result.cmdDescriptors, 3)
	for _, cmd := range result.cmdDescriptors {
		ctxValue := reflect.ValueOf(*cmd.command).FieldByName("ctx")
		assert.False(t, ctxValue.IsNil())
		assert.True(t, ctxValue.Equal(reflect.ValueOf(t.Context())), "each command should have the same context")
	}
}
