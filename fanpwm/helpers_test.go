package fanpwm

import (
	"errors"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"
)

var _ wrOnlyFile = (*fakeFile)(nil)

type ffArgPassedToTruncate struct {
	val int64
	ts  time.Time
}

type ffArgPassedToWrite struct {
	val []byte
	ts  time.Time
}

type ffArgPassedToSeek struct {
	ts     time.Time
	off    int64
	whence int
}

type fakeFile struct {
	actualTruncates []ffArgPassedToTruncate
	onTruncateErrs  []error
	actualWrites    []ffArgPassedToWrite
	onWriteErrs     []error
	actualSeeks     []ffArgPassedToSeek
	onSeekErrs      []error
	onCloseErrs     []error
	mutex           sync.Mutex
}

func (ff *fakeFile) Close() error {
	ff.mutex.Lock()
	defer ff.mutex.Unlock()

	if len(ff.onCloseErrs) == 0 {
		return nil
	}
	err := ff.onCloseErrs[0]
	ff.onCloseErrs = ff.onCloseErrs[1:]
	return err
}

func (ff *fakeFile) Truncate(sz int64) (err error) {
	ff.mutex.Lock()
	defer ff.mutex.Unlock()

	ts := time.Now()
	if len(ff.onTruncateErrs) > 0 {
		err = ff.onTruncateErrs[0]
		ff.onTruncateErrs = ff.onTruncateErrs[1:]
	}
	passedArg := ffArgPassedToTruncate{val: sz, ts: ts}
	ff.actualTruncates = append(ff.actualTruncates, passedArg)
	return
}

func (ff *fakeFile) Write(b []byte) (n int, err error) {
	ff.mutex.Lock()
	defer ff.mutex.Unlock()

	ts := time.Now()
	if len(ff.onWriteErrs) > 0 {
		err = ff.onWriteErrs[0]
		ff.onWriteErrs = ff.onWriteErrs[1:]
	}
	passedArg := ffArgPassedToWrite{val: b, ts: ts}
	ff.actualWrites = append(ff.actualWrites, passedArg)
	return
}

func (ff *fakeFile) Seek(off int64, whence int) (n int64, err error) {
	ff.mutex.Lock()
	defer ff.mutex.Unlock()

	ts := time.Now()
	if len(ff.onSeekErrs) > 0 {
		err = ff.onSeekErrs[0]
		ff.onSeekErrs = ff.onSeekErrs[1:]
	}
	passedArg := ffArgPassedToSeek{off: off, whence: whence, ts: ts}
	ff.actualSeeks = append(ff.actualSeeks, passedArg)
	return
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

func testDriver(t *testing.T) (*Driver, *fakeFile) {
	t.Helper()

	tmpFile, cleanupTmpFile := temporaryFile(t)
	defer cleanupTmpFile()

	driver, err := New(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	devFile := new(fakeFile)
	driver.devFile = devFile

	return driver, devFile
}

func iter(n int) []struct{} {
	return make([]struct{}, n)
}
