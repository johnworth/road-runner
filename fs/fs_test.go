package fs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"testing"
)

type testFS struct {
	filemap    map[string]*byteFile
	failOpen   bool
	failCreate bool
	failRemove bool
	failFile   bool
}

func newTestFS() *testFS {
	return &testFS{
		filemap: map[string]*byteFile{},
	}
}

type byteFile struct {
	fail bool
	*bytes.Buffer
}

func newByteFile(fail bool) *byteFile {
	return &byteFile{
		fail,
		bytes.NewBuffer([]byte{}),
	}
}

func (b *byteFile) Close() error {
	return nil
}

func (b *byteFile) Read(p []byte) (int, error) {
	if b.fail {
		return 0, errors.New("read error")
	}
	return b.Buffer.Read(p)
}

func (b *byteFile) Write(p []byte) (int, error) {
	if b.fail {
		return 0, errors.New("write error")
	}
	return b.Buffer.Write(p)
}

func (t *testFS) Open(path string) (File, error) {
	if t.failOpen {
		return nil, errors.New("failOpen was true")
	}
	if _, ok := t.filemap[path]; !ok {
		return nil, errors.New("failed to open file")
	}
	return t.filemap[path], nil
}

func (t *testFS) Create(path string) (File, error) {
	if t.failCreate {
		return nil, errors.New("failed to create file")
	}
	t.filemap[path] = newByteFile(t.failFile)
	return t.filemap[path], nil
}

func (t *testFS) Remove(path string) error {
	if t.failRemove {
		return errors.New("failed to remove file")
	}
	delete(t.filemap, path)
	return nil
}

func TestCopyJobFile(t *testing.T) {
	tfs := newTestFS()
	uuid := "00000000-0000-0000-0000-000000000000"
	from := path.Join("test", fmt.Sprintf("%s.json", uuid))
	c, err := tfs.Create(from)
	if err != nil {
		t.Error(err)
	}
	c.Write([]byte("this is a test"))
	to := "/tmp"
	err = CopyJobFile(tfs, uuid, from, to)
	if err != nil {
		t.Error(err)
	}
	tmpPath := path.Join(to, fmt.Sprintf("%s.json", uuid))
	if _, err = tfs.Open(tmpPath); err != nil {
		t.Error(err)
	}

	// test failures from the Open() function
	tfs = newTestFS()
	tfs.failOpen = true
	err = CopyJobFile(tfs, uuid, from, to)
	if err == nil {
		t.Error(err)
	}

	// test failures from the Create() function.
	tfs = newTestFS()
	c, err = tfs.Create(from)
	if err != nil {
		t.Error(err)
	}
	c.Write([]byte("this is a test"))
	tfs.failCreate = true
	err = CopyJobFile(tfs, uuid, from, to)
	if err == nil {
		t.Error(err)
	}

	// test failures from the Copy() function.
	tfs = newTestFS()
	c, err = tfs.Create(from)
	if err != nil {
		t.Error(err)
	}
	c.Write([]byte("this is a test"))
	tfs.failFile = true
	err = CopyJobFile(tfs, uuid, from, to)
	if err == nil {
		t.Error(err)
	}
}

func TestDeleteJobFile(t *testing.T) {
	tfs := newTestFS()
	uuid := "00000000-0000-0000-0000-000000000000"
	from := path.Join("test", fmt.Sprintf("%s.json", uuid))
	c, err := tfs.Create(from)
	if err != nil {
		t.Error(err)
	}
	c.Write([]byte("this is a test"))
	to := "/tmp"
	err = CopyJobFile(tfs, uuid, from, to)
	if err != nil {
		t.Error(err)
	}
	DeleteJobFile(tfs, uuid, to)
	tmpPath := path.Join(to, fmt.Sprintf("%s.json", uuid))
	if _, err := os.Open(tmpPath); err == nil {
		t.Errorf("tmpPath %s existed after deleteJobFile() was called", tmpPath)
	}
}

func TestDeleteJobFileFail(t *testing.T) {
	tfs := newTestFS()
	tfs.failRemove = true
	uuid := "00000000-0000-0000-0000-000000000000"
	from := path.Join("test", fmt.Sprintf("%s.json", uuid))
	c, err := tfs.Create(from)
	if err != nil {
		t.Error(err)
	}
	c.Write([]byte("this is a test"))
	to := "/tmp"
	err = CopyJobFile(tfs, uuid, from, to)
	if err != nil {
		t.Error(err)
	}
	err = DeleteJobFile(tfs, uuid, to)
	if err == nil {
		t.Error("err was nil")
	}
}
