package cmdchain

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"testing"
)

func TestIgnoreExitCode(t *testing.T) {
	err := exec.Command(testHelper, "-x", "13").Run()

	assert.False(t, IgnoreExitCode(13)(0, nil, err))
	assert.True(t, IgnoreExitCode(1)(0, nil, err))
}

func TestIgnoreExitErrors(t *testing.T) {
	err := exec.Command(testHelper, "-x", "13").Run()

	assert.False(t, IgnoreExitErrors()(0, nil, err))
	assert.True(t, IgnoreExitErrors()(0, nil, fmt.Errorf("someOtherError")))
}

func TestIgnoreAll(t *testing.T) {
	err := exec.Command(testHelper, "-x", "13").Run()

	assert.False(t, IgnoreAll()(0, nil, err))
	assert.False(t, IgnoreAll()(0, nil, fmt.Errorf("someOtherError")))
}

func TestIgnoreNothing(t *testing.T) {
	err := exec.Command(testHelper, "-x", "13").Run()

	assert.True(t, IgnoreNothing()(0, nil, err))
	assert.True(t, IgnoreNothing()(0, nil, fmt.Errorf("someOtherError")))
}
