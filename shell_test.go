package cmdchain

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestFromShell(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		expectedString string
		expectError    string
	}{
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
	}

	for _, tt := range tests {
		t.Run(t.Name()+"_"+tt.name, func(t *testing.T) {
			cmd, err := FromShell(tt.command)

			if tt.expectError == "" {
				assert.NoError(t, err)
				assert.Equal(t, strings.TrimSpace(tt.expectedString), strings.TrimSpace(cmd.String()))
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
			}
		})
	}
}
