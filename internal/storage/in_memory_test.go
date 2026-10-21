package storage

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestAddDirs(t *testing.T) {
	store := NewInMemoryStorage()
	if err := store.HelperAddDir("/dir1/dir2/dir3"); err != nil {
		t.Fatalf("AddDir failed, %v", err)
	}
	dir1 := store.root.children["dir1"]
	if dir1 == nil || dir1.nodeType != NodeTypeDirectory {
		t.Fatalf("FindNode returned nil or a file for existing directory")
	}
	dir2 := dir1.children["dir2"]
	if dir2 == nil || dir2.nodeType != NodeTypeDirectory {
		t.Fatalf("FindNode returned nil or a file for existing directory")
	}
	dir3 := dir2.children["dir3"]
	if dir3 == nil || dir3.nodeType != NodeTypeDirectory {
		t.Fatalf("FindNode returned nil or a file for existing directory")
	}
}

func TestCreateFile(t *testing.T) {
	store := NewInMemoryStorage()
	if err := store.HelperAddDir("/dir1/dir2"); err != nil {
		t.Fatalf("AddDir failed, %v", err)
	}
	err := store.Create("/dir1/dir2/file.txt", 0644)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	meta, err := store.Stat("/dir1/dir2/file.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if meta.Type != NodeTypeFile {
		t.Errorf("type = %q, want %q", meta.Type, NodeTypeFile)
	}
	if meta.Mode != 0644 {
		t.Errorf("mode = %#o, want 0644", meta.Mode)
	}
	if meta.Size != 0 {
		t.Errorf("size = %d, want 0", meta.Size)
	}
}

var invalidPaths = []struct {
	path string
}{
	{""},
	{"file.txt"},
	{"/"},
	{"/dir/"},
	{"/dir//file.txt"},
}

func TestCreateErrInvalidPath(t *testing.T) {
	for _, tt := range invalidPaths {
		t.Run(tt.path, func(t *testing.T) {
			store := NewInMemoryStorage()
			err := store.Create(tt.path, 0644)
			if !errors.Is(err, ErrInvalidPath) {
				t.Fatalf("Create should fail on invalid path: %v", err)
			}
		})
	}
}

func TestCreateErrNotDirectory(t *testing.T) {
	store := NewInMemoryStorage()
	if err := store.HelperAddDir("/dir1"); err != nil {
		t.Fatalf("AddDir failed, %v", err)
	}
	err := store.Create("/dir1/dir2", 0777)
	if err != nil {
		t.Fatalf("Create file failed: %v", err)
	}
	err = store.Create("/dir1/dir2/file.txt", 0644)
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("Create should fail on creating file inside file: %v", err)
	}
}

func TestCreateErrAlreadyExists(t *testing.T) {
	store := NewInMemoryStorage()

	workers := 100
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			err := store.Create("/file.txt", 0777)
			errs <- err
		}()
	}

	close(start)
	wg.Wait()
	close(errs)
	createdCount := 0
	for err := range errs {
		if err == nil {
			createdCount++
			continue
		}
		if !errors.Is(err, ErrAlreadyExists) {
			t.Fatalf("Crate should fail on creating creating when file already exists: %v", err)
		}
	}
	if createdCount != 1 {
		t.Fatalf("Expected file to created exactly once but got: %d", createdCount)
	}
}

func TestStatDir(t *testing.T) {
	store := setupTestStorage(t)
	meta, err := store.Stat("/")

	if err != nil {
		t.Fatalf("Stat returned error")
	}

	if meta.Type != NodeTypeDirectory {
		t.Fatalf("MetaData type mismatch: got %q, want %q", meta.Type, NodeTypeDirectory)
	}

	if meta.NLink != 4 {
		t.Fatalf("MetaData Nlink mismatch: got %d, want %d", meta.NLink, 4)
	}
	if meta.Size != 0 {
		t.Fatalf("MetaData Size mismatch: got %d, want %d", meta.Size, 0)
	}
	if meta.Mode != 0755 {
		t.Fatalf("MetaData Mode mismatch: got %#o, want %#o", meta.Mode, 0755)
	}
}

func TestStatFile(t *testing.T) {
	store := setupTestStorage(t)
	meta, err := store.Stat("/swap")

	if err != nil {
		t.Fatalf("Stat returned error: got %q, want nil", err)
	}

	if meta.Type != NodeTypeFile {
		t.Fatalf("MetaData type mismatch: got %q, want %q", meta.Type, NodeTypeFile)
	}

	if meta.NLink != 1 {
		t.Fatalf("MetaData Nlink mismatch: got %d, want %d", meta.NLink, 1)
	}
	if meta.Size != 16 {
		t.Fatalf("MetaData Size mismatch: got %d, want %d", meta.Size, 16)
	}
	if meta.Mode != 0644 {
		t.Fatalf("MetaData Mode mismatch: got %#o, want %#o", meta.Mode, 0644)
	}
}

func TestStatNotExisting(t *testing.T) {
	store := setupTestStorage(t)
	_, err := store.Stat("/not-existing")

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Stat expected to error, got %q, want %q", err, ErrNotFound)
	}

}

func TestStatInvalidPath(t *testing.T) {
	store := setupTestStorage(t)
	_, err := store.Stat("/swap//file.txt")

	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Stat expected to error, got %q, want %q", err, ErrInvalidPath)
	}
}

func TestListValidPath(t *testing.T) {
	store := setupTestStorage(t)
	entries, err := store.List("/home")
	expected := []Entry{
		{Name: "others"}, //dirs first
		{Name: "damian"}, //then alphabethical order
		{Name: "julia"},
	}

	if err != nil {
		t.Fatalf("List returned error: got %q, want nil", err)
	}
	if len(expected) != len(entries) {
		t.Fatalf("List returned different entries: got %d, want %d", len(entries), len(expected))
	}
	for i, entry := range entries {
		if entry != expected[i] {
			t.Fatalf("Entry at position %d mismatch: got %q, want %q", i, entry.Name, expected[i].Name)
		}
	}
}

func TestListOnFile(t *testing.T) {
	store := setupTestStorage(t)
	_, err := store.List("/swap")

	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("List expected to error, got %q, want %q", err, ErrNotDirectory)
	}
}

func TestReadAt(t *testing.T) {
	store := setupTestStorage(t)
	buff := make([]byte, 8)
	n, err := store.ReadAt("/home/damian", buff, 6)

	if err != nil {
		t.Fatalf("ReadAt returned error: got %q, want nil", err)
	}

	if n != 8 {
		t.Fatalf("ReadAt should return bytes written: got %d, want %d", n, 8)
	}

	if string(buff) != "home, my" {
		t.Fatalf("Expected to read: got %s, want %s", buff, "home, my")
	}
}

func TestReadAtEnd(t *testing.T) {
	store := setupTestStorage(t)
	buff := make([]byte, 10)
	n, err := store.ReadAt("/home/damian", buff, 15)

	if err != nil {
		t.Fatalf("ReadAt returned error: got %q, want nil", err)
	}

	if n != 4 {
		t.Fatalf("ReadAt should return bytes written: got %d, want %d", n, 4)
	}

	if string(buff[:n]) != "home" {
		t.Fatalf("Expected to read: got %s, want %s", buff, "home")
	}
}

func TestReadAtEOF(t *testing.T) {
	store := NewInMemoryStorage()
	if err := store.HelperAddFile("/file.txt", []byte("hello")); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	tests := []struct {
		name   string
		offset int64
	}{
		{name: "exact EOF", offset: 5},
		{name: "past EOF", offset: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := store.ReadAt("/file.txt", make([]byte, 4), tt.offset)
			if err != nil {
				t.Fatalf("ReadAt returned error: %v", err)
			}
			if n != 0 {
				t.Errorf("ReadAt returned %d bytes, want 0", n)
			}
		})
	}
}

func TestReadAtNegativeOffset(t *testing.T) {
	store := NewInMemoryStorage()
	if err := store.HelperAddFile("/file.txt", []byte("hello")); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	_, err := store.ReadAt("/file.txt", make([]byte, 1), -1)
	if !errors.Is(err, ErrInvalidOffset) {
		t.Fatalf("ReadAt error = %v, want ErrInvalidOffset", err)
	}
}

func TestReadAtDirectory(t *testing.T) {
	store := NewInMemoryStorage()
	if err := store.HelperAddDir("/dir"); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

	_, err := store.ReadAt("/dir", make([]byte, 1), 0)
	if !errors.Is(err, ErrIsDirectory) {
		t.Fatalf("ReadAt error = %v, want ErrIsDirectory", err)
	}
}

func TestWriting(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		change  string
		ofst    int64
		want    string
	}{
		{"write to empty file", "", "hello", 0, "hello"},
		{"write at the start", "hello", "X", 0, "Xello"},
		{"write in the middle", "hello", "XY", 1, "hXYlo"},
		{"write at the end", "hello", "hello", 5, "hellohello"},
		{"write after the end", "hello", "hello", 8, "hello\x00\x00\x00hello"},
		{"write empty at the start", "hello", "", 0, "hello"},
		{"write empty in the middle", "hello", "", 3, "hello"},
		{"write empty at the end", "hello", "", 5, "hello"},
		{"write empty past the end", "hello", "", 8, "hello"},
		{"write 1 byte at the start", "hello", "X", 0, "Xello"},
		{"write 1 byte in the middle", "hello", "X", 2, "heXlo"},
		{"write 1 byte past the end", "hello", "X", 8, "hello\x00\x00\x00X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/test.txt"
			store := setupTestStorage(t)
			if err := store.Create(path, 0644); err != nil {
				t.Fatalf("Created failed %v", err)
			}
			if len(tt.initial) > 0 {
				if _, err := store.WriteAt(path, []byte(tt.initial), 0); err != nil {
					t.Fatalf("WriteAt inital data failed: %v", err)
				}
			}
			written, err := store.WriteAt(path, []byte(tt.change), tt.ofst)
			if err != nil {
				t.Fatalf("WriteAt change data failed: %v", err)
			}
			if written != len(tt.change) {
				t.Fatalf("WriteAt wrote partially: %d of %d", written, len(tt.change))
			}

			buff := make([]byte, 100)
			read, err := store.ReadAt(path, buff, 0)
			if err != nil {
				t.Fatalf("Unexpected error while reading, %v", err)
			}
			actual := string(buff[0:read])
			if actual != tt.want {
				t.Errorf("Write stored %q, want %q", actual, tt.want)
			}

			meta, err := store.Stat(path)
			if err != nil {
				t.Fatalf("Unexpected error while stat, %v", err)
			}

			if meta.Size != int64(len(tt.want)) {
				t.Errorf("Written stored file size %d, want %d", meta.Size, len(tt.want))
			}
		})
	}
}

func TestWritingNegOffset(t *testing.T) {
	path := "/test.txt"
	store := setupTestStorage(t)
	if err := store.Create(path, 0644); err != nil {
		t.Fatalf("Create failed %v", err)
	}

	_, err := store.WriteAt(path, []byte("any"), -1)

	if !errors.Is(err, ErrInvalidOffset) {
		t.Fatalf("Write got %v, want %v", err, ErrInvalidOffset)
	}
}

func TestWritingErrIsDirectory(t *testing.T) {
	path := "/test"
	store := setupTestStorage(t)
	if err := store.HelperAddDir(path); err != nil {
		t.Fatalf("CreateDir failed %v", err)
	}

	_, err := store.WriteAt(path, []byte("any"), 0)

	if !errors.Is(err, ErrIsDirectory) {
		t.Fatalf("Write got %v, want %v", err, ErrIsDirectory)
	}
}

func TestWritingErrNotFound(t *testing.T) {
	path := "/test"
	store := setupTestStorage(t)

	_, err := store.WriteAt(path, []byte("any"), 0)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Write got %v, want %v", err, ErrNotFound)
	}
}

func TestConcurrentListAndAddFile(t *testing.T) {
	store := NewInMemoryStorage()
	if err := store.HelperAddDir("/files"); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

	const (
		writerCount = 4
		readerCount = 4
		iterations  = 4
	)

	start := make(chan struct{})
	errs := make(chan error, (writerCount+readerCount)*iterations)

	var wg sync.WaitGroup

	for writer := range writerCount {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			for iteration := range iterations {
				path := fmt.Sprintf(
					"/files/file-%d-%d.txt",
					writer,
					iteration,
				)

				if err := store.HelperAddFile(path, []byte("content")); err != nil {
					errs <- fmt.Errorf("AddFile(%q): %w", path, err)
				}
			}
		}()
	}

	for range readerCount {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			for range iterations {
				if _, err := store.List("/files"); err != nil {
					errs <- fmt.Errorf("List: %w", err)
				}
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}

	entries, err := store.List("/files")
	if err != nil {
		t.Fatalf("final List failed: %v", err)
	}

	want := writerCount * iterations
	if len(entries) != want {
		t.Errorf("final entry count = %d, want %d", len(entries), want)
	}

}

func setupTestStorage(t *testing.T) *InMemoryStorage {
	t.Helper()

	store := NewInMemoryStorage()
	if err := store.HelperAddDir("/home"); err != nil {
		t.Fatalf("Failed in test setup: %v", err)
	}
	if err := store.HelperAddFile("/home/julia", []byte("art is key")); err != nil {
		t.Fatalf("Failed in test setup: %v", err)
	}
	if err := store.HelperAddFile("/home/damian", []byte("smeet home, my home")); err != nil {
		t.Fatalf("Failed in test setup: %v", err)
	}
	if err := store.HelperAddDir("/home/others"); err != nil {
		t.Fatalf("Failed in test setup: %v", err)
	}

	if err := store.HelperAddDir("/tmp"); err != nil {
		t.Fatalf("Failed in test setup: %v", err)
	}
	if err := store.HelperAddFile("/swap", []byte("some random data")); err != nil {
		t.Fatalf("Failed in test setup: %v", err)
	}
	return store
}
