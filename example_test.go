package cmdchain_test

import (
	"bytes"
	"context"
	"github.com/rainu/go-command-chain"
	"os"
	"os/exec"
	"strings"
	"time"
)

func ExampleBuilder() {
	output := &bytes.Buffer{}

	//it's the same as in shell: ls -l | grep README | wc -l
	err := cmdchain.Builder().
		Join("ls", "-l").
		Join("grep", "README").
		Join("wc", "-l").
		Finalize().
		WithOutput(output).
		Run()

	if err != nil {
		panic(err)
	}
	println(output.String())
}

func ExampleBuilder_join() {
	//it's the same as in shell: ls -l | grep README
	err := cmdchain.Builder().
		Join("ls", "-l").
		Join("grep", "README").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_finalize() {
	//it's the same as in shell: ls -l | grep README
	err := cmdchain.Builder().
		Join("ls", "-l").
		Join("grep", "README").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_joinCmd() {
	//it's the same as in shell: ls -l | grep README
	grepCmd := exec.Command("grep", "README")

	//do NOT manipulate the command's streams!

	err := cmdchain.Builder().
		Join("ls", "-l").
		JoinCmd(grepCmd).
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_joinWithContext() {
	//the "ls" command will be killed after 1 second
	ctx, cancelFn := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFn()

	//it's the same as in shell: ls -l | grep README
	err := cmdchain.Builder().
		JoinWithContext(ctx, "ls", "-l").
		Join("grep", "README").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_joinShellCmd() {
	//it's the same as in shell: ls -l | grep README
	err := cmdchain.Builder().
		JoinShellCmd(`ls -l | grep README`).
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_withInput() {
	inputContent := strings.NewReader("test\n")

	//it's the same as in shell: echo "test" | grep test
	err := cmdchain.Builder().
		WithInput(inputContent).
		Join("grep", "test").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_forwardError() {
	//it's the same as in shell: echoErr "test" |& grep test
	err := cmdchain.Builder().
		Join("echoErr", "test").ForwardError().
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_discardStdOut() {
	//this will drop the stdout from echo .. so grep will receive no input
	//Attention: it must be used in combination with ForwardError - otherwise
	//it will cause a invalid stream configuration error!
	err := cmdchain.Builder().
		Join("echo", "test").DiscardStdOut().ForwardError().
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_withOutputForks() {
	//it's the same as in shell: echo "test" | tee <fork> | grep test | wc -l
	outputFork := &bytes.Buffer{}

	err := cmdchain.Builder().
		Join("echo", "test").WithOutputForks(outputFork).
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
	println(outputFork.String())
}

func ExampleBuilder_withErrorForks() {
	//it's the same as in shell: echoErr "test" |& tee <fork> | grep test | wc -l
	errorFork := &bytes.Buffer{}

	err := cmdchain.Builder().
		Join("echoErr", "test").ForwardError().WithErrorForks(errorFork).
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
	println(errorFork.String())
}

func ExampleBuilder_withInjections() {
	//it's the same as in shell: echo -e "test\ntest" | grep test | wc -l
	inputContent := strings.NewReader("test\n")

	err := cmdchain.Builder().
		Join("echoErr", "test").WithInjections(inputContent).
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_withAdditionalEnvironment() {
	//it's the same as in shell: TEST=VALUE TEST2=2 env | grep TEST | wc -l
	err := cmdchain.Builder().
		Join("env").WithAdditionalEnvironment("TEST", "VALUE", "TEST2", 2).
		Join("grep", "TEST").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_withOutput() {
	//it's the same as in shell: echo "test" | grep test > /tmp/output

	target, err := os.OpenFile("/tmp/output", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}

	err = cmdchain.Builder().
		Join("echo", "test").
		Join("grep", "test").
		Finalize().WithOutput(target).Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_withError() {
	//it's the same as in shell: echoErr "test" 2> /tmp/error

	target, err := os.OpenFile("/tmp/error", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}

	err = cmdchain.Builder().
		Join("echoErr", "test").
		Finalize().WithError(target).Run()

	if err != nil {
		panic(err)
	}
}

func ExampleBuilder_run() {
	output := &bytes.Buffer{}

	//it's the same as in shell: ls -l | grep README | wc -l
	err := cmdchain.Builder().
		Join("ls", "-l").
		Join("grep", "README").
		Join("wc", "-l").
		Finalize().
		WithOutput(output).
		Run()

	if err != nil {
		panic(err)
	}
	println(output.String())
}

func ExampleBuilder_runAndGet() {
	//it's the same as in shell: ls -l | grep README | wc -l
	sout, serr, err := cmdchain.Builder().
		Join("ls", "-l").
		Join("grep", "README").
		Join("wc", "-l").
		Finalize().
		RunAndGet()

	if err != nil {
		panic(err)
	}
	println("OUTPUT: " + sout)
	println("ERROR: " + serr)
}
