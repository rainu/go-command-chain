package cmdchain

import (
	"os"
)

// lazyFile is a wrapper around os.File that lazily opens the file when the first write operation is performed.
type lazyFile struct {
	name string
	flag int
	perm os.FileMode

	file    *os.File
	fileErr error
}

func newLazyFile(name string, flag int, perm os.FileMode) *lazyFile {
	return &lazyFile{
		name: name,
		flag: flag,
		perm: perm,
	}
}

func (l *lazyFile) Write(p []byte) (n int, err error) {
	l.BeforeRun()

	if l.fileErr != nil {
		return 0, l.fileErr
	}

	return l.file.Write(p)
}

func (l *lazyFile) BeforeRun() {
	if l.file == nil {
		l.file, l.fileErr = os.OpenFile(l.name, l.flag, l.perm)
	}
}

func (l *lazyFile) AfterRun() {
	l.Close()
}

func (l *lazyFile) Close() (err error) {
	if l.file != nil {
		err = l.file.Close()

		// reset to nil to ensure it is reopened on next write operation
		l.file = nil
		l.fileErr = nil
	}

	return
}

func (l *lazyFile) String() string {
	if l.flag&os.O_APPEND != 0 {
		return l.name + " (appending)"
	}
	return l.name
}
