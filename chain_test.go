package cmdchain

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"
)

var testHelper string

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	testHelper = path.Join(wd, "testHelper")

	//build a little go binary which can be executed and process some stdOut/stdErr output
	err = exec.Command("go", "build", "-ldflags", "-w -s", "-o", testHelper, "./test_helper/main.go").Run()
	if err != nil {
		panic(err)
	}
}

func TestSimple(t *testing.T) {
	toTest := Builder().
		Join("ls", "-l").
		Join("grep", "README").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")
}

func TestSimple_apply(t *testing.T) {
	toTest := Builder().
		Join(testHelper, "-pwd").Apply(func(_ int, command *exec.Cmd) {
		command.Dir = os.TempDir()
	})

	runAndCompare(t, toTest, os.TempDir()+"\n")
}

func TestCombined_applyBeforeStart(t *testing.T) {
	outViaBuilder := &bytes.Buffer{}
	outViaApplier := &bytes.Buffer{}

	Builder().
		Join("echo", "test").ApplyBeforeStart(func(_ int, cmd *exec.Cmd) {
		assert.Same(t, outViaBuilder, cmd.Stdout)
		cmd.Stdout = outViaApplier
	}).
		Finalize().WithOutput(outViaBuilder).Run()

	assert.Equal(t, "", outViaBuilder.String())
	assert.Equal(t, "test\n", outViaApplier.String())
}

func TestSimple_stderr(t *testing.T) {
	output := &bytes.Buffer{}

	err := Builder().
		Join(testHelper, "-e", "ERROR", "-o", "TEST").
		Finalize().WithError(output).Run()

	assert.NoError(t, err)
	assert.Equal(t, "ERROR\n", output.String())
}

func TestSimple_multi_stdout(t *testing.T) {
	output1 := &bytes.Buffer{}
	output2 := &bytes.Buffer{}

	err := Builder().
		Join(testHelper, "-e", "ERROR", "-o", "TEST").
		Finalize().WithOutput(output1, output2).Run()

	assert.NoError(t, err)
	assert.Equal(t, output1.String(), output2.String())
}

func TestSimple_multi_stderr(t *testing.T) {
	output1 := &bytes.Buffer{}
	output2 := &bytes.Buffer{}

	err := Builder().
		Join(testHelper, "-e", "ERROR", "-o", "TEST").
		Finalize().WithError(output1, output2).Run()

	assert.NoError(t, err)
	assert.Equal(t, output1.String(), output2.String())
}

func TestSimple_WithInput(t *testing.T) {
	toTest := Builder().
		WithInput(strings.NewReader("TEST\nOUTPUT")).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")
}

func TestSimple_WithMultiInput(t *testing.T) {
	toTest := Builder().
		WithInput(strings.NewReader("TEST\nOUTPUT"), strings.NewReader("TEST\n")).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "2\n")
}

func TestSimple_WithEnvironment(t *testing.T) {
	toTest := Builder().
		Join(testHelper, "-pe").WithEnvironment("TEST", "VALUE", "TEST2", 2)

	runAndCompare(t, toTest, "TEST=VALUE\nTEST2=2\n")
}

func TestSimple_WithEnvironmentMap(t *testing.T) {
	toTest := Builder().
		Join(testHelper, "-pe").WithEnvironmentMap(map[interface{}]interface{}{"TEST": "VALUE", "TEST2": 2})

	runAndCompare(t, toTest, "TEST=VALUE\nTEST2=2\n")
}

func TestSimple_WithAdditionalEnvironment(t *testing.T) {
	toTest := Builder().
		Join(testHelper, "-pe").WithAdditionalEnvironment("TEST", "VALUE", "TEST2", 2).
		Join("grep", "TEST").
		Join("sort")

	runAndCompare(t, toTest, "TEST2=2\nTEST=VALUE\n")
}

func TestSimple_WithAdditionalEnvironmentMap(t *testing.T) {
	toTest := Builder().
		Join(testHelper, "-pe").WithAdditionalEnvironmentMap(map[interface{}]interface{}{"TEST": "VALUE", "TEST2": 2}).
		Join("grep", "TEST").
		Join("sort")

	runAndCompare(t, toTest, "TEST2=2\nTEST=VALUE\n")
}

func TestSimple_WithAdditionalEnvironment_butNotProcessEnv(t *testing.T) {
	cmd := exec.Command(testHelper, "-pe")
	cmd.Env = []string{"TEST=VALUE"}

	toTest := Builder().
		JoinCmd(cmd).WithAdditionalEnvironment("TEST2", 2)

	runAndCompare(t, toTest, "TEST=VALUE\nTEST2=2\n")
}

func TestSimple_WithAdditionalEnvironmentMap_butNotProcessEnv(t *testing.T) {
	cmd := exec.Command(testHelper, "-pe")
	cmd.Env = []string{"TEST=VALUE"}

	toTest := Builder().
		JoinCmd(cmd).WithAdditionalEnvironmentMap(map[interface{}]interface{}{"TEST2": 2})

	runAndCompare(t, toTest, "TEST=VALUE\nTEST2=2\n")
}

func TestSimple_WithEnvironment_InvalidArguments(t *testing.T) {
	err := Builder().
		Join(testHelper, "-pe").WithEnvironment("TEST", "VALUE", "TEST2").
		Finalize().Run()

	assert.Error(t, err)
	assert.Equal(t, "one or more chain build errors occurred: [0 - invalid count of environment arguments]", err.Error())
}

func TestSimple_WithWorkingDirectory(t *testing.T) {
	toTest := Builder().
		Join(testHelper, "-pwd").WithWorkingDirectory(os.TempDir())

	runAndCompare(t, toTest, os.TempDir()+"\n")
}

func TestCombined(t *testing.T) {
	output := &bytes.Buffer{}

	err := Builder().
		Join(testHelper, "-to", "100ms", "-te", "100ms", "-ti", "1ms").ForwardError().
		Join("grep", `OUT\|ERR`).
		Finalize().WithOutput(output).Run()

	assert.NoError(t, err)

	assert.Contains(t, output.String(), "OUT\nERR\nOUT\n", "It seams that the streams will not processed parallel!")
}

func TestCombined_forked(t *testing.T) {
	output := &bytes.Buffer{}
	outFork := &bytes.Buffer{}
	errFork := &bytes.Buffer{}

	err := Builder().
		Join(testHelper, "-to", "100ms", "-te", "100ms", "-ti", "1ms").ForwardError().WithOutputForks(outFork).WithErrorForks(errFork).
		Join("grep", `OUT\|ERR`).
		Finalize().WithOutput(output).Run()

	assert.NoError(t, err)

	assert.Contains(t, output.String(), "OUT\nERR\nOUT\n", "It seams that the streams will not processed parallel!")
	assert.Contains(t, outFork.String(), "OUT\nOUT\n")
	assert.NotContains(t, outFork.String(), "ERR\n")
	assert.Contains(t, errFork.String(), "ERR\nERR\n")
	assert.NotContains(t, errFork.String(), "OUT\n")
}

func TestWithContext(t *testing.T) {
	output := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), 110*time.Millisecond)
	defer cancel()

	err := Builder().
		JoinWithContext(ctx, testHelper, "-to", "1s", "-ti", "100ms").
		Join("grep", `OUT\|ERR`).
		Finalize().WithOutput(output).Run()

	assert.Error(t, err)
	assert.Equal(t, "one or more command has returned an error: [0 - signal: killed; 1 - ]", err.Error())

	assert.Equal(t, "OUT\n", output.String(), "It seams that the process was not interrupted.")
}

func TestSimple_ErrorForked(t *testing.T) {
	output := &bytes.Buffer{}

	toTest := Builder().
		Join(testHelper, "-e", "ERROR", "-o", "TEST").WithErrorForks(output).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	assert.Contains(t, output.String(), "ERROR", "The error of 'testHelper' seams not to be forked!")
}

func TestStdErr_OnlyError(t *testing.T) {
	toTest := Builder().
		Join(testHelper, "-e", "TEST").DiscardStdOut().ForwardError().
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")
}

func TestStdErr_OnlyErrorButOutForked(t *testing.T) {
	output := &bytes.Buffer{}

	toTest := Builder().
		Join(testHelper, "-e", "TEST", "-o", "TEST_OUT").DiscardStdOut().WithOutputForks(output).ForwardError().
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	assert.Contains(t, output.String(), "TEST_OUT", "The output of 'testHelper' seams not to be forked!")
}

func TestOutputFork_single(t *testing.T) {
	output := &bytes.Buffer{}

	toTest := Builder().
		Join("ls", "-l").
		Join("grep", "README").WithOutputForks(output).
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	assert.Contains(t, output.String(), "README.md", "The output of 'ls -l' seams not to be forked!")
}

func TestOutputFork_multiple(t *testing.T) {
	output1 := &bytes.Buffer{}
	output2 := &bytes.Buffer{}

	toTest := Builder().
		Join("ls", "-l").
		Join("grep", "README").WithOutputForks(output1, output2).
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	assert.Equal(t, output1.String(), output2.String(), "The output seams not to be forked to both forks!")
}

func TestErrorFork_single(t *testing.T) {
	output := &bytes.Buffer{}

	toTest := Builder().
		Join(testHelper, "-e", "TEST").DiscardStdOut().ForwardError().WithErrorForks(output).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	assert.Equal(t, output.String(), "TEST\n", "The error of './testHelper' seams not to be forked!")
}

func TestErrorFork_multiple(t *testing.T) {
	output1 := &bytes.Buffer{}
	output2 := &bytes.Buffer{}

	toTest := Builder().
		Join(testHelper, "-e", "TEST").DiscardStdOut().ForwardError().WithErrorForks(output1, output2).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	assert.Equal(t, output1.String(), output2.String(), "The error seams not to be forked to both forks!")
}

func TestInputInjection(t *testing.T) {
	toTest := Builder().
		Join(testHelper, "-o", "TEST").
		Join("grep", "TEST").
		WithInjections(strings.NewReader("TEST\n")).
		Join("wc", "-l")

	runAndCompare(t, toTest, "2\n")
}

func TestInputInjectionWithoutStdin(t *testing.T) {
	input := strings.NewReader("Hello")
	output := bytes.NewBuffer([]byte{})
	err := Builder().
		Join("cat").WithInjections(input).
		Finalize().
		WithOutput(output).
		Run()

	assert.NoError(t, err)
	assert.Equal(t, "Hello", output.String())
}

func TestInvalidStreamLink(t *testing.T) {
	err := Builder().
		Join("ls", "-l").DiscardStdOut().
		Join("grep", "TEST").
		Join("wc", "-l").
		Finalize().Run()

	assert.Error(t, err)
	mError := err.(MultipleErrors)
	assert.Equal(t, "invalid stream configuration", mError.Errors()[0].Error())
}

func TestBrokenStream(t *testing.T) {
	out, _ := os.CreateTemp("", ".txt")
	defer os.Remove(out.Name())

	//close the file so the stream can not be written -> this should cause a stream error!
	out.Close()

	err := Builder().
		Join("ls", "-l").WithOutputForks(out).
		Join("grep", "README").
		Join("wc", "-l").
		Finalize().Run()

	assert.Error(t, err)
	mError := err.(MultipleErrors)
	assert.Contains(t, mError.Errors()[1].Error(), "file already closed")
}

func TestInvalidCommand(t *testing.T) {
	err := Builder().
		Join("ls", "-l").
		Join("invalidApplication").
		Finalize().Run()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start command")
}

func TestBrokenStreamAndRunError(t *testing.T) {
	out, _ := os.CreateTemp("", ".txt")
	defer os.Remove(out.Name())

	//close the file so the stream can not be written -> this should cause a stream error!
	out.Close()

	err := Builder().
		Join("ls", "-l").WithOutputForks(out).
		Join("grep", "aslnaslkdnan").
		Finalize().Run()

	assert.Error(t, err)
	mError := err.(MultipleErrors)
	assert.Equal(t, 2, len(mError.Errors()))
	assert.Contains(t, mError.Errors()[0].Error(), "one or more command has returned an error")
	assert.Contains(t, mError.Errors()[1].Error(), "one or more command stream copies failed")
}

func TestIgnoreExitCode(t *testing.T) {
	err := Builder().
		Join(testHelper, "-o", "test", "-x", "1").WithErrorChecker(IgnoreExitCode(1)).
		Join("grep", "test").
		Finalize().Run()

	assert.NoError(t, err)
}

func runAndCompare(t *testing.T, toTest CommandBuilder, expected string) {
	output := &bytes.Buffer{}

	err := toTest.Finalize().WithOutput(output).Run()
	assert.NoError(t, err)
	assert.Equal(t, expected, output.String())
}
