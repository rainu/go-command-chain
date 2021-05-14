package cmdchain

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

func ExampleBuilder() {
	output := &bytes.Buffer{}

	//it's the same as in shell: ls -l | grep README | wc -l
	err := Builder().
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

func ExampleChainBuilder_Join() {
	//it's the same as in shell: ls -l | grep README
	err := Builder().
		Join("ls", "-l").
		Join("grep", "README").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleChainBuilder_Finalize() {
	//it's the same as in shell: ls -l | grep README
	err := Builder().
		Join("ls", "-l").
		Join("grep", "README").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleChainBuilder_JoinCmd() {
	//it's the same as in shell: ls -l | grep README
	grepCmd := exec.Command("grep", "README")

	//do NOT manipulate the command's streams!

	err := Builder().
		Join("ls", "-l").
		JoinCmd(grepCmd).
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleChainBuilder_JoinWithContext() {
	//the "ls" command will be killed after 1 second
	ctx, cancelFn := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFn()

	//it's the same as in shell: ls -l | grep README
	err := Builder().
		JoinWithContext(ctx, "ls", "-l").
		Join("grep", "README").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleFirstCommandBuilder_WithInput() {
	inputContent := strings.NewReader("test\n")

	//it's the same as in shell: echo "test" | grep test
	err := Builder().
		WithInput(inputContent).
		Join("grep", "test").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleCommandBuilder_ForwardError() {
	//it's the same as in shell: echoErr "test" |& grep test
	err := Builder().
		Join("echoErr", "test").ForwardError().
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleCommandBuilder_DiscardStdOut() {
	//this will drop the stdout from echo .. so grep will receive no input
	//Attention: it must be used in combination with ForwardError - otherwise
	//it will cause a invalid stream configuration error!
	err := Builder().
		Join("echo", "test").DiscardStdOut().ForwardError().
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleCommandBuilder_WithOutputForks() {
	//it's the same as in shell: echo "test" | tee <fork> | grep test | wc -l
	outputFork := &bytes.Buffer{}

	err := Builder().
		Join("echo", "test").WithOutputForks(outputFork).
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
	println(outputFork.String())
}

func ExampleCommandBuilder_WithErrorForks() {
	//it's the same as in shell: echoErr "test" |& tee <fork> | grep test | wc -l
	errorFork := &bytes.Buffer{}

	err := Builder().
		Join("echoErr", "test").ForwardError().WithErrorForks(errorFork).
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
	println(errorFork.String())
}

func ExampleCommandBuilder_WithInjections() {
	//it's the same as in shell: echo -e "test\ntest" | grep test | wc -l
	inputContent := strings.NewReader("test\n")

	err := Builder().
		Join("echoErr", "test").WithInjections(inputContent).
		Join("grep", "test").
		Join("wc", "-l").
		Finalize().Run()

	if err != nil {
		panic(err)
	}
}

func ExampleFinalizedBuilder_WithOutput() {
	//it's the same as in shell: echo "test" | grep test > /tmp/output

	target, err := os.OpenFile("/tmp/output", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}

	err = Builder().
		Join("echo", "test").
		Join("grep", "test").
		Finalize().WithOutput(target).Run()

	if err != nil {
		panic(err)
	}
}

func ExampleFinalizedBuilder_WithError() {
	//it's the same as in shell: echoErr "test" 2> /tmp/error

	target, err := os.OpenFile("/tmp/error", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}

	err = Builder().
		Join("echoErr", "test").
		Finalize().WithError(target).Run()

	if err != nil {
		panic(err)
	}
}

func ExampleFinalizedBuilder_Run() {
	output := &bytes.Buffer{}

	//it's the same as in shell: ls -l | grep README | wc -l
	err := Builder().
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
