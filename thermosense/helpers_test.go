package thermosense

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"
)

var _ rdOnlyFile = (*fakeFile)(nil)

type fakeFile struct {
	onSeekErrs  []error
	onReadErrs  []error
	onCloseErrs []error
}

func (ff *fakeFile) Close() error {
	if len(ff.onCloseErrs) == 0 {
		return nil
	}
	err := ff.onCloseErrs[0]
	ff.onCloseErrs = ff.onCloseErrs[1:]
	return err
}

func (ff *fakeFile) Seek(_ int64, _ int) (_ int64, err error) {
	if len(ff.onSeekErrs) > 0 {
		err = ff.onSeekErrs[0]
		ff.onSeekErrs = ff.onSeekErrs[1:]
	}
	return
}

func (ff *fakeFile) Read(b []byte) (_ int, err error) {
	if len(ff.onReadErrs) > 0 {
		err = ff.onReadErrs[0]
		ff.onReadErrs = ff.onReadErrs[1:]
	}
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

func iter(n int) []struct{} {
	return make([]struct{}, n)
}
