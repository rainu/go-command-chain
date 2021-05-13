package command_chain

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

func TestSimple_WithInput(t *testing.T) {
	toTest := Builder().
		WithInput(strings.NewReader("TEST\nOUTPUT")).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")
}

func TestCombined(t *testing.T) {
	output := &bytes.Buffer{}

	err := Builder().
		Join(testHelper, "-to", "100ms", "-te", "100ms", "-ti", "1ms").ForwardError().
		Join("grep", `OUT\|ERR`).
		Finalize().WithOutput(output).Run()

	assert.NoError(t, err)

	assert.NotContains(t, output.String(), "OUT\nOUT\nOUT\nOUT\nOUT\nOUT\nOUT", "It seams that the streams will not processed parallel!")
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
		Join(testHelper, "-e", "TEST").BlockingOutput().ForwardError().
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")
}

func TestStdErr_OnlyErrorButOutForked(t *testing.T) {
	output := &bytes.Buffer{}

	toTest := Builder().
		Join(testHelper, "-e", "TEST", "-o", "TEST_OUT").BlockingOutput().WithOutputForks(output).ForwardError().
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
		Join(testHelper, "-e", "TEST").BlockingOutput().ForwardError().WithErrorForks(output).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	assert.Equal(t, output.String(), "TEST\n", "The error of './testHelper' seams not to be forked!")
}

func TestErrorFork_multiple(t *testing.T) {
	output1 := &bytes.Buffer{}
	output2 := &bytes.Buffer{}

	toTest := Builder().
		Join(testHelper, "-e", "TEST").BlockingOutput().ForwardError().WithErrorForks(output1, output2).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	assert.Equal(t, output1.String(), output2.String(), "The error seams not to be forked to both forks!")
}

func TestInvalidStreamLink(t *testing.T) {
	err := Builder().
		Join("ls", "-l").BlockingOutput().
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
