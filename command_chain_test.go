package go_command_chain

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func TestName(t *testing.T) {
	c1 := exec.Command("ls", "-l")
	c2 := exec.Command("grep", "README")
	c3 := exec.Command("wc", "-l")

	c2.Stdin, _ = c1.StdoutPipe()
	c3.Stdin, _ = c2.StdoutPipe()
	c3.Stdout = os.Stdout

	assert.NoError(t, c1.Start())
	assert.NoError(t, c2.Start())
	assert.NoError(t, c3.Start())

	assert.NoError(t, c1.Wait())
	assert.NoError(t, c2.Wait())
	assert.NoError(t, c3.Wait())
}

func TestName1(t *testing.T) {

	tmpFile, err := os.CreateTemp("", "output")
	assert.NoError(t, err)

	err = Builder().
		Join("ls", "-l").
		WithOutputForks(tmpFile).
		Join("grep", "README").
		Join("wc", "-l").
		Finalize().
		WithOutput(os.Stdout).
		Run()

	assert.NoError(t, err)

	fmt.Println(ioutil.ReadFile(tmpFile.Name()))
}
