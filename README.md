[![Go](https://github.com/rainu/go-command-chain/actions/workflows/build.yml/badge.svg)](https://github.com/rainu/go-command-chain/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/rainu/go-command-chain/branch/main/graph/badge.svg)](https://codecov.io/gh/rainu/go-command-chain)
[![Go Report Card](https://goreportcard.com/badge/github.com/rainu/go-command-chain)](https://goreportcard.com/report/github.com/rainu/go-command-chain)
[![Go Reference](https://pkg.go.dev/badge/github.com/rainu/go-command-chain.svg)](https://pkg.go.dev/github.com/rainu/go-command-chain)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

# go-command-chain
![](https://media.discordapp.net/attachments/1101609055094575192/1102605830592925717/rainu_cyberpunkt_netrunner_cat_1_dad93401-86aa-4ea2-b4d6-c171077d401d.png)

A go library for easy configure and run command chains. Such like pipelining in unix shells.

# Example
```sh
cat log_file.txt | grep error | wc -l
```

```go
package main

import (
	"fmt"
	"github.com/rainu/go-command-chain"
)

func main() {
	stdOut, stdErr, err := cmdchain.Builder().
		Join("cat", "log_file.txt").
		Join("grep", "error").
		Join("wc", "-l").
		Finalize().RunAndGet()

	if err != nil {
		panic(err)
	}
	if stdErr != "" {
		panic(stdErr)
	}
	fmt.Printf("Errors found: %s", stdOut)
}
```

```go
package main

import (
	"fmt"
	"github.com/rainu/go-command-chain"
)

func main() {
	stdOut, stdErr, err := cmdchain.Builder().
		JoinShellCmd(`cat log_file.txt | grep error | wc -l`).
		Finalize().RunAndGet()

	if err != nil {
		panic(err)
	}
	if stdErr != "" {
		panic(stdErr)
	}
	fmt.Printf("Errors found: %s", stdOut)
}
```

For more examples how to use the command chain see [examples](example_test.go).

# Why you should use this library?

If you want to execute a complex command pipeline you could come up with the idea of just execute **one** command: the
shell itself such like to following code:

```go
package main

import (
	"os/exec"
)

func main() {
	exec.Command("sh", "-c", "cat log_file.txt | grep error | wc -l").Run()
}
```

But this procedure has some negative points:
* you must have installed the shell - in correct version - on the system itself
    * so you are dependent on the shell
* you have no control over the individual commands - only the parent process (shell command itself)
* pipelining can be complex (redirection of stderr etc.) - so you have to know the pipeline syntax
    * maybe this syntax is different for shell versions

## (advanced) features

### input injections
**Multiple** different input stream for each command can be configured. This can be useful if you want to 
forward multiple input sources to one command.

```go
package main

import (
	"github.com/rainu/go-command-chain"
	"strings"
)

func main() {
	inputContent1 := strings.NewReader("content from application itself\n")
	inputContent2 := strings.NewReader("another content from application itself\n")

	err := cmdchain.Builder().
		Join("echo", "test").WithInjections(inputContent1, inputContent2).
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}
```

### forking of stdout and stderr

Stdout and stderr of **each** command can be **forked** to different io.Writer.

```go
package main

import (
	"bytes"
	"github.com/rainu/go-command-chain"
)

func main() {
	echoErr := &bytes.Buffer{}
	echoOut := &bytes.Buffer{}
	grepErr := &bytes.Buffer{}
	
	err := cmdchain.Builder().
		Join("echo", "test").WithOutputForks(echoOut).WithErrorForks(echoErr).
		Join("grep", "test").WithErrorForks(grepErr).
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}
```
