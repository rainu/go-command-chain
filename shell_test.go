package cmdchain

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"os/exec"
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
			name:    "error redirection",
			command: `date 2> /tmp/err | grep 'Hello'`,
			expectedString: `
[SO]               ╭╮                       ╿
[CM] /usr/bin/date ╡╰ /usr/bin/grep "Hello" ╡
[SE]               │                        ╽
[ES]               ╰  /tmp/err
`,
		},
		{
			name:    "error redirection (appending)",
			command: `date 2>> /tmp/err | grep 'Hello'`,
			expectedString: `
[SO]               ╭╮                       ╿
[CM] /usr/bin/date ╡╰ /usr/bin/grep "Hello" ╡
[SE]               │                        ╽
[ES]               ╰  /tmp/err (appending)
`,
		},
		{
			name:    "file redirection",
			command: `date > /tmp/out |& grep 'Hello'`,
			expectedString: `
[OS]               ╭  /tmp/out
[SO]               ├╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ╰╯                       ╽
`,
			check: func(t *testing.T, c *chain) {
				assert.Empty(t, c.cmdDescriptors[0].errorStreams)
			},
		},
		{
			name:    "file redirection (appending)",
			command: `date >> /tmp/out |& grep 'Hello'`,
			expectedString: `
[OS]               ╭  /tmp/out (appending)
[SO]               ├╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ╰╯                       ╽
`,
			check: func(t *testing.T, c *chain) {
				assert.Empty(t, c.cmdDescriptors[0].errorStreams)
			},
		},
		{
			name:    "file redirection (error)",
			command: `date 2> /tmp/out |& grep 'Hello'`,
			expectedString: `
[SO]               ╭╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ├╯                       ╽
[ES]               ╰  /tmp/out
`,
		},
		{
			name:    "file redirection (error appending)",
			command: `date 2>> /tmp/out |& grep 'Hello'`,
			expectedString: `
[SO]               ╭╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ├╯                       ╽
[ES]               ╰  /tmp/out (appending)
`,
		},
		{
			name:    "both file redirection",
			command: `date &> /tmp/out |& grep 'Hello'`,
			expectedString: `
[OS]               ╭  /tmp/out
[SO]               ├╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ├╯                       ╽
[ES]               ╰  /tmp/out
`,
		},
		{
			name:    "both file redirection (appending)",
			command: `date &>> /tmp/out |& grep 'Hello'`,
			expectedString: `
[OS]               ╭  /tmp/out (appending)
[SO]               ├╮                       ╿
[CM] /usr/bin/date ╡╞ /usr/bin/grep "Hello" ╡
[SE]               ├╯                       ╽
[ES]               ╰  /tmp/out (appending)
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildShellChain(Builder().JoinShellCmd(tt.command).(*shellChain)).(*chain)

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
	result := buildShellChain(Builder().JoinShellCmdWithContext(t.Context(), `echo "hello world" | grep "hello" | wc -c`).(*shellChain)).(*chain)

	assert.False(t, result.buildErrors.hasError)
	assert.Len(t, result.cmdDescriptors, 3)
	for _, cmd := range result.cmdDescriptors {
		ctxValue := reflect.ValueOf(*cmd.command).FieldByName("ctx")
		assert.False(t, ctxValue.IsNil())
		assert.True(t, ctxValue.Equal(reflect.ValueOf(t.Context())), "each command should have the same context")
	}
}

func TestJoinShellCmdAndWithAdditionalForks(t *testing.T) {
	output := bytes.NewBuffer(nil)
	finalized := Builder().
		JoinShellCmd(`echo "hello world" | grep "hello" | wc -c`).
		WithOutputForks(output).
		Finalize()

	expectedString := `
[OS]                             ╭  *bytes.Buffer         ╭  *bytes.Buffer    ╭ *bytes.Buffer
[SO]                             ├╮                       ├╮                  │
[CM] /usr/bin/echo "hello world" ╡╰ /usr/bin/grep "hello" ╡╰ /usr/bin/wc "-c" ╡
[SE]                             ╽                        ╽                   ╽
`
	assert.Equal(t, strings.TrimSpace(expectedString), strings.TrimSpace(finalized.String()))

	sOut, sErr, err := finalized.RunAndGet()
	assert.NoError(t, err)
	assert.Equal(t, "12", strings.TrimSpace(sOut))
	assert.Equal(t, "", strings.TrimSpace(sErr))
}

func TestJoinShellCmdAndCommandApplier(t *testing.T) {
	var applied []int
	Builder().
		Join("echo", "hello world").
		JoinShellCmd(`grep "hello" | wc -c`).
		Apply(func(index int, command *exec.Cmd) {
			applied = append(applied, index)
		}).
		Finalize()

	assert.Equal(t, []int{1, 2}, applied)
}

func TestShellActionCollection(t *testing.T) {
	testCases := []struct {
		name   string
		mock   func(*MockCommandBuilder)
		action func(*shellChain)
	}{
		{"Apply",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().Apply(gomock.Any())
			},
			func(s *shellChain) {
				s.Apply(func(int, *exec.Cmd) {})
			},
		},
		{"ApplyBeforeStart",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().ApplyBeforeStart(gomock.Any())
			},
			func(s *shellChain) {
				s.ApplyBeforeStart(func(int, *exec.Cmd) {})
			},
		},
		{"ForwardError",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().ForwardError()
			},
			func(s *shellChain) {
				s.ForwardError()
			},
		},
		{"DiscardStdOut",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().DiscardStdOut()
			},
			func(s *shellChain) {
				s.DiscardStdOut()
			},
		},
		{"WithOutputForks",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithOutputForks()
			},
			func(s *shellChain) {
				s.WithOutputForks()
			},
		},
		{"WithAdditionalOutputForks",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithAdditionalOutputForks()
			},
			func(s *shellChain) {
				s.WithAdditionalOutputForks()
			},
		},
		{"WithErrorForks",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithErrorForks()
			},
			func(s *shellChain) {
				s.WithErrorForks()
			},
		},
		{"WithAdditionalErrorForks",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithAdditionalErrorForks()
			},
			func(s *shellChain) {
				s.WithAdditionalErrorForks()
			},
		},
		{"WithInjections",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithInjections()
			},
			func(s *shellChain) {
				s.WithInjections()
			},
		},
		{"WithEmptyEnvironment",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithEmptyEnvironment()
			},
			func(s *shellChain) {
				s.WithEmptyEnvironment()
			},
		},
		{"WithEnvironment",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithEnvironment()
			},
			func(s *shellChain) {
				s.WithEnvironment()
			},
		},
		{"WithEnvironmentMap",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithEnvironmentMap(gomock.Any())
			},
			func(s *shellChain) {
				s.WithEnvironmentMap(map[interface{}]interface{}{})
			},
		},
		{"WithEnvironmentPairs",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithEnvironmentPairs()
			},
			func(s *shellChain) {
				s.WithEnvironmentPairs()
			},
		},
		{"WithAdditionalEnvironment",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithAdditionalEnvironment()
			},
			func(s *shellChain) {
				s.WithAdditionalEnvironment()
			},
		},
		{"WithAdditionalEnvironmentMap",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithAdditionalEnvironmentMap(gomock.Any())
			},
			func(s *shellChain) {
				s.WithAdditionalEnvironmentMap(map[interface{}]interface{}{})
			},
		},
		{"WithAdditionalEnvironmentPairs",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithAdditionalEnvironmentPairs()
			},
			func(s *shellChain) {
				s.WithAdditionalEnvironmentPairs()
			},
		},
		{"WithWorkingDirectory",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithWorkingDirectory(gomock.Any())
			},
			func(s *shellChain) {
				s.WithWorkingDirectory("")
			},
		},
		{"WithErrorChecker",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().WithErrorChecker(gomock.Any())
			},
			func(s *shellChain) {
				s.WithErrorChecker(nil)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mChain := NewMockCommandBuilder(ctrl)
			tt.mock(mChain)

			toTest := &shellChain{}
			tt.action(toTest)

			for _, action := range toTest.actions {
				action(mChain)
			}
		})
	}
}

func TestShellBuildBeforeNext(t *testing.T) {
	testCases := []struct {
		name   string
		mock   func(*MockCommandBuilder)
		action func(*shellChain)
	}{
		{"Join",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().Join("echo")
			},
			func(s *shellChain) {
				s.Join("echo")
			},
		},
		{"JoinCmd",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().JoinCmd(nil)
			},
			func(s *shellChain) {
				s.JoinCmd(nil)
			},
		},
		{"JoinWithContext",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().JoinWithContext(t.Context(), "echo")
			},
			func(s *shellChain) {
				s.JoinWithContext(t.Context(), "echo")
			},
		},
		{"JoinShellCmd",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().JoinShellCmd("echo")
			},
			func(s *shellChain) {
				s.JoinShellCmd("echo")
			},
		},
		{"JoinShellCmdWithContext",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().JoinShellCmdWithContext(t.Context(), "echo")
			},
			func(s *shellChain) {
				s.JoinShellCmdWithContext(t.Context(), "echo")
			},
		},
		{"Finalize",
			func(builder *MockCommandBuilder) {
				builder.EXPECT().Finalize()
			},
			func(s *shellChain) {
				s.Finalize()
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			of := buildShellChain
			defer func() { buildShellChain = of }()

			mChain := NewMockCommandBuilder(ctrl)
			buildShellChain = func(s *shellChain) CommandBuilder {
				return mChain
			}

			tt.mock(mChain)
			toTest := &shellChain{}
			tt.action(toTest)
		})
	}

}
