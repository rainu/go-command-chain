package cmdchain

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path"
	"testing"
)

func TestLazyFile(t *testing.T) {
	toTest := newLazyFile(path.Join(t.TempDir(), "lazy_file_test"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	s, _ := os.Stat(toTest.name)
	assert.Nil(t, s, "file should not exist")

	toTest.BeforeRun()
	s, _ = os.Stat(toTest.name)
	assert.NotNil(t, s, "file should exist")

	_, err := toTest.Write([]byte("first write"))
	require.NoError(t, err)
	toTest.AfterRun()

	f, err := os.Open(toTest.name)
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	assert.NoError(t, err)
	assert.Equal(t, "first write", string(content))

	// second write should reopen the file
	toTest.BeforeRun()
	_, err = toTest.Write([]byte("second write"))
	require.NoError(t, err)
	toTest.AfterRun()

	f.Seek(0, 0)
	content, err = io.ReadAll(f)
	assert.NoError(t, err)
	assert.Equal(t, "second write", string(content))
}
