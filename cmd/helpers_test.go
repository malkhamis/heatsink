package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func backupProcArgs(t *testing.T) (restore func()) {
	t.Helper()
	orig := make([]string, len(os.Args))
	copy(orig, os.Args)
	return func() {
		os.Args = orig
	}
}

// stdoutStream pipes stdout so it can be read line by line using the the returned channel.
// Read errors, other than io.EOF, are pushed to the streamErr channel. On errors, including
// io.EOF, both channels are closed. It is the caller's responsibility to call restoreStdout
// to bring os.Stdout back to its previous state before calling this function
func stdoutStream(t *testing.T) (stdoutLines <-chan []byte, streamErr <-chan error, restoreStdout func()) {
	t.Helper()

	originalStdout := os.Stdout
	pipeRd, pipeWr, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating os pipe: %v", err)
	}
	os.Stdout = pipeWr
	restoreStdout = func() { os.Stdout = originalStdout }

	stdoutLinesChan := make(chan []byte)
	streamErrChan := make(chan error)

	go func() {
		defer close(streamErrChan)
		defer close(stdoutLinesChan)

		for logStream := bufio.NewReader(pipeRd); ; {
			next, cerr := logStream.ReadBytes('\n')
			if cerr == io.EOF {
				stdoutLinesChan <- next
				return
			}
			if cerr != nil {
				streamErrChan <- fmt.Errorf("error reading stdout stream's next line: %w", cerr)
				return
			}
			stdoutLinesChan <- next
		}
	}()

	return stdoutLinesChan, streamErrChan, restoreStdout
}

func temporaryFile(t *testing.T) (file *os.File, cleanup func()) {
	t.Helper()

	tmpFile, err := ioutil.TempFile("", t.Name()+"-")
	if err != nil {
		t.Fatal(err)
	}

	cleanup = func() {
		err := tmpFile.Close()
		if err != nil && !errors.Is(err, os.ErrClosed) {
			t.Logf("%s: error closing a temporary test file: %s", tmpFile.Name(), err)
		}
		err = os.Remove(tmpFile.Name())
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Logf("%s: error removing temporary test file: %s", tmpFile.Name(), err)
		}
	}

	return tmpFile, cleanup
}
