package fusefs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dwilkolek/rake-fs/internal/storage"
	"github.com/winfsp/cgofuse/fuse"
)

func TestGetattrRoot(t *testing.T) {
	_, fs := createTestFs()
	stat := &fuse.Stat_t{}

	result := fs.Getattr("/", stat, 0)

	if result != 0 {
		t.Fatalf("Getattr(/) returned %d, want 0", result)
	}
	if stat.Mode&fuse.S_IFMT != fuse.S_IFDIR {
		t.Errorf("root is not a directory: mode = %#o", stat.Mode)
	}
	if stat.Mode&0755 != 0555 {
		t.Errorf("permissions = %#o, want 0555", stat.Mode&0555)
	}
	if stat.Nlink != 2 {
		t.Errorf("Nlink = %d, want 2", stat.Nlink)
	}
}

func TestGetattrMissingPath(t *testing.T) {
	_, fs := createTestFs()
	stat := &fuse.Stat_t{}

	result := fs.Getattr("/missing", stat, 0)

	if result != -fuse.ENOENT {
		t.Errorf("Getattr(/missing) returned %d, want %d",
			result, -fuse.ENOENT)
	}
}

func TestReaddirRoot(t *testing.T) {
	_, fs := createTestFs()
	var names []string

	result := fs.Readdir(
		"/",
		func(name string, stat *fuse.Stat_t, offset int64) bool {
			names = append(names, name)
			return true
		},
		0,
		0,
	)

	if result != 0 {
		t.Fatalf("Readdir(/) returned %d, want 0", result)
	}

	if len(names) != 2 || names[0] != "." || names[1] != ".." {
		t.Errorf("entries = %v, want [. ..]", names)
	}
}

func TestReadWholeFile(t *testing.T) {
	store, fs := createTestFs()
	expectedContent := "This is a test file."
	store.AddFile("/testfile.txt", []byte(expectedContent))

	var buff = make([]byte, len(expectedContent))
	var readBytes = fs.Read("/testfile.txt", buff, 0, 0)

	if readBytes != len(expectedContent) {
		t.Errorf("Read returned %d bytes, want %d", readBytes, len(expectedContent))
	}
	if string(buff) != expectedContent {
		t.Errorf("testfile.txt data = %q, want %q", string(buff[:readBytes]), expectedContent)
	}
}

func TestOpenFile(t *testing.T) {
	store, fs := createTestFs()
	store.AddFile("/testfile.txt", []byte("This is a test file."))

	if errc, _ := fs.Open("/testfile.txt", fuse.O_RDONLY); errc != fuse.F_OK {
		t.Errorf("Open returned %d, want %d", errc, fuse.F_OK)
	}
	if errc, _ := fs.Open("/testfile.txt", fuse.O_WRONLY); errc != -fuse.EACCES {
		t.Errorf("Open returned %d, want %d", errc, -fuse.EACCES)
	}
	if errc, _ := fs.Open("/testfile.txt", fuse.O_RDWR); errc != -fuse.EACCES {
		t.Errorf("Open returned %d, want %d", errc, -fuse.EACCES)
	}
}

func TestOpenNotExistingFileOrDir(t *testing.T) {
	store, fs := createTestFs()
	store.AddDir("test")

	if errc, _ := fs.Open("/test/test.txt", fuse.O_RDONLY); errc != -fuse.ENOENT {
		t.Errorf("Open returned %d, want %d", errc, -fuse.ENOENT)
	}

	if errc, _ := fs.Open("/dir", fuse.O_RDONLY); errc != -fuse.ENOENT {
		t.Errorf("Open returned %d, want %d", errc, -fuse.ENOENT)
	}

}

func TestReadFile(t *testing.T) {
	expectedContent := "is"
	store, fs := createTestFs()
	store.AddFile("/testfile.txt", []byte("This is a test file."))

	var buff = make([]byte, 2)
	var readBytes = fs.Read("/testfile.txt", buff, 5, 0)

	if readBytes != 2 {
		t.Errorf("Read returned %d bytes, want 2", readBytes)
	}
	if string(buff) != expectedContent {
		t.Errorf("testfile.txt data = %q, want %q", string(buff), expectedContent)
	}
}

func TestReadDir(t *testing.T) {
	store, fs := createTestFs()
	store.AddDir("/testdir")

	if status := fs.Read("/testdir", make([]byte, 2), 5, 0); status != -fuse.EISDIR {
		t.Errorf("Read returned %d bytes, want status %d", status, -fuse.EISDIR)
	}

	if status := fs.Read("/not_existing", make([]byte, 2), 5, 0); status != -fuse.ENOENT {
		t.Errorf("Read returned %d bytes, want status %d", status, -fuse.ENOENT)
	}
}

func TestReaddirStopsWhenFillReturnsFalse(t *testing.T) {
	_, fs := createTestFs()
	var names []string

	result := fs.Readdir("/", func(
		name string,
		stat *fuse.Stat_t,
		offset int64,
	) bool {
		names = append(names, name)
		return false
	}, 0, 0)

	if result != 0 {
		t.Fatalf("Readdir returned %d, want 0", result)
	}
	if len(names) != 1 {
		t.Fatalf("entries = %v, want only one entry", names)
	}
}

func TestReaddirFileReturnsENOTDIR(t *testing.T) {
	store, fs := createTestFs()

	if err := store.AddFile("/file.txt", nil); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	result := fs.Readdir(
		"/file.txt",
		func(string, *fuse.Stat_t, int64) bool {
			return true
		},
		0,
		0,
	)

	if result != -fuse.ENOTDIR {
		t.Errorf("Readdir returned %d, want %d", result, -fuse.ENOTDIR)
	}
}

func TestReadDirectoryReturnsEISDIR(t *testing.T) {
	store, fs := createTestFs()
	store.AddDir("/dir")

	result := fs.Read("/dir", make([]byte, 10), 0, 0)

	if result != -fuse.EISDIR {
		t.Errorf("Read returned %d, want %d", result, -fuse.EISDIR)
	}
}

func TestReadNegativeOffsetReturnsEINVAL(t *testing.T) {
	store, fs := createTestFs()
	store.AddFile("/file.txt", []byte("hello"))

	result := fs.Read("/file.txt", make([]byte, 2), -1, 0)

	if result != -fuse.EINVAL {
		t.Errorf("Read returned %d, want %d", result, -fuse.EINVAL)
	}
}

func TestStorageErrno(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"not found", storage.ErrNotFound, -fuse.ENOENT},
		{"not directory", storage.ErrNotDirectory, -fuse.ENOTDIR},
		{"is directory", storage.ErrIsDirectory, -fuse.EISDIR},
		{"invalid path", storage.ErrInvalidPath, -fuse.EINVAL},
		{"invalid offset", storage.ErrInvalidOffset, -fuse.EINVAL},
		{"unexpected", errors.New("backend failed"), -fuse.EIO},
		{"wrapped not found", fmt.Errorf("stat /missing: %w", storage.ErrNotFound), -fuse.ENOENT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := storageErrno(tt.err)
			if got != tt.want {
				t.Errorf("storageErrno() = %d, want %d", got, tt.want)
			}
		})
	}
}

func createTestFs() (*storage.InMemoryStorage, *FileSystem) {
	store := storage.NewInMemoryStorage()
	fs := NewFileSystem(store)
	return store, fs
}
