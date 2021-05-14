package cmdchain

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const testHelper = "./testHelper"

func init() {
	//build a little go binary which can be executed and process some stdOut/stdErr output
	err := exec.Command("go", "build", "-ldflags", "-w -s", "-o", testHelper, "./test_helper/main.go").Run()
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

func runAndCompare(t *testing.T, toTest CommandBuilder, expected string) {
	output := &bytes.Buffer{}

	err := toTest.Finalize().WithOutput(output).Run()
	assert.NoError(t, err)
	assert.Equal(t, expected, output.String())
}
