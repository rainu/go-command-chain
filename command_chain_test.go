package go_command_chain

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
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
	tmpFile := mkTmp(t)

	err := Builder().
		Join(testHelper, "-to", "100ms", "-te", "100ms", "-ti", "1ms").
		ForwardError().
		Join("grep", `OUT\|ERR`).
		Finalize().WithOutput(tmpFile).Run()

	assert.NoError(t, err)

	content := readContent(t, tmpFile)
	assert.NotContains(t, content, "OUT\nOUT\nOUT\nOUT", "It seams that the streams will not processed parallel!")
}

func TestSimple_ErrorForked(t *testing.T) {
	tmpFile := mkTmp(t)

	toTest := Builder().
		Join(testHelper, "-e", "ERROR", "-o", "TEST").
		WithErrorForks(tmpFile).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	content := readContent(t, tmpFile)
	assert.Contains(t, content, "ERROR", "The error of 'testHelper' seams not to be forked!")
}

func TestStdErr_OnlyError(t *testing.T) {
	toTest := Builder().
		Join(testHelper, "-e", "TEST").
		BlockingOutput().
		ForwardError().
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")
}

func TestStdErr_OnlyErrorButOutForked(t *testing.T) {
	tmpFile := mkTmp(t)

	toTest := Builder().
		Join(testHelper, "-e", "TEST", "-o", "TEST_OUT").
		BlockingOutput().
		WithOutputForks(tmpFile).
		ForwardError().
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	content := readContent(t, tmpFile)
	assert.Contains(t, content, "TEST_OUT", "The output of 'testHelper' seams not to be forked!")
}

func TestOutputFork_single(t *testing.T) {
	tmpFile := mkTmp(t)

	toTest := Builder().
		Join("ls", "-l").
		Join("grep", "README").
		WithOutputForks(tmpFile).
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	content := readContent(t, tmpFile)
	assert.Contains(t, content, "README.md", "The output of 'ls -l' seams not to be forked!")
}

func TestOutputFork_multiple(t *testing.T) {
	tmpFile1 := mkTmp(t)
	tmpFile2 := mkTmp(t)

	toTest := Builder().
		Join("ls", "-l").
		Join("grep", "README").
		WithOutputForks(tmpFile1, tmpFile2).
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	content1 := readContent(t, tmpFile1)
	content2 := readContent(t, tmpFile2)

	assert.Equal(t, content1, content2, "The output seams not to be forked to both forks!")
}

func TestErrorFork_single(t *testing.T) {
	tmpFile := mkTmp(t)

	toTest := Builder().
		Join(testHelper, "-e", "TEST").
		BlockingOutput().
		ForwardError().
		WithErrorForks(tmpFile).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	content := readContent(t, tmpFile)
	assert.Equal(t, content, "TEST\n", "The error of './testHelper' seams not to be forked!")
}

func TestErrorFork_multiple(t *testing.T) {
	tmpFile1 := mkTmp(t)
	tmpFile2 := mkTmp(t)

	toTest := Builder().
		Join(testHelper, "-e", "TEST").
		BlockingOutput().
		ForwardError().
		WithErrorForks(tmpFile1, tmpFile2).
		Join("grep", "TEST").
		Join("wc", "-l")

	runAndCompare(t, toTest, "1\n")

	content1 := readContent(t, tmpFile1)
	content2 := readContent(t, tmpFile2)
	assert.Equal(t, content1, content2, "The error seams not to be forked to both forks!")
}

func TestInvalidStreamLink(t *testing.T) {
	err := Builder().
		Join("ls", "-l").
		BlockingOutput().
		Join("grep", "TEST").
		Join("wc", "-l").
		Finalize().Run()

	assert.Error(t, err)
	mError := err.(MultipleErrors)
	assert.Equal(t, "invalid stream configuration", mError.Errors()[0].Error())
}

func runAndCompare(t *testing.T, toTest CommandBuilder, expected string) {
	tmpFile := mkTmp(t)
	err := toTest.Finalize().WithOutput(tmpFile).Run()
	assert.NoError(t, err)

	content := readContent(t, tmpFile)
	assert.Equal(t, expected, content)
}

func mkTmp(t *testing.T) *os.File {
	tmpFile, err := os.CreateTemp("", "output")
	assert.NoError(t, err)

	if err != nil {
		panic(err)
	}

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile
}

func readContent(t *testing.T, file *os.File) string {
	content, err := ioutil.ReadFile(file.Name())
	assert.NoError(t, err)

	if err != nil {
		panic(err)
	}
	return string(content)
}
