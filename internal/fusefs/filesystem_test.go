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
	if stat.Mode&0777 != 0755 {
		t.Errorf("permissions = %#o, want 0755", stat.Mode&0777)
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
	if err := store.HelperAddFile("/testfile.txt", []byte(expectedContent)); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

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
	tests := []struct {
		name      string
		filePerm  int
		readOnly  int
		writeOnly int
		readWrite int
	}{
		{"read write", 0600, fuse.F_OK, fuse.F_OK, fuse.F_OK},
		{"read only", 0400, fuse.F_OK, -fuse.EACCES, -fuse.EACCES},
		{"write only", 0200, -fuse.EACCES, fuse.F_OK, -fuse.EACCES},
		{"no access", 0000, -fuse.EACCES, -fuse.EACCES, -fuse.EACCES},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, fs := createTestFs()
			file := "/testfile.txt"
			if err := store.Create(file, tt.filePerm); err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			if errc, _ := fs.Open(file, fuse.O_RDONLY); errc != tt.readOnly {
				t.Errorf("Open returned %d, want %d", errc, tt.readOnly)
			}
			if errc, _ := fs.Open(file, fuse.O_WRONLY); errc != tt.writeOnly {
				t.Errorf("Open returned %d, want %d", errc, tt.writeOnly)
			}
			if errc, _ := fs.Open(file, fuse.O_RDWR); errc != tt.readWrite {
				t.Errorf("Open returned %d, want %d", errc, tt.readWrite)
			}
		})
	}

}

func TestOpenDirectory(t *testing.T) {
	storage, fs := createTestFs()

	if err := storage.HelperAddDir("/dir"); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

	if errc, _ := fs.Open("/dir", fuse.O_RDONLY); errc != -fuse.EISDIR {
		t.Errorf("Open returned %d, want %d", errc, -fuse.EISDIR)
	}
}

func TestOpenMissingFile(t *testing.T) {
	store, fs := createTestFs()
	if err := store.HelperAddDir("/test"); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

	if errc, _ := fs.Open("/test/test.txt", fuse.O_RDONLY); errc != -fuse.ENOENT {
		t.Errorf("Open returned %d, want %d", errc, -fuse.ENOENT)
	}

	if errc, _ := fs.Open("/test2", fuse.O_RDONLY); errc != -fuse.ENOENT {
		t.Errorf("Open returned %d, want %d", errc, -fuse.ENOENT)
	}
}

func TestReadFile(t *testing.T) {
	expectedContent := "is"
	store, fs := createTestFs()
	if err := store.HelperAddFile("/testfile.txt", []byte("This is a test file.")); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

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
	if err := store.HelperAddDir("/testdir"); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

	if status := fs.Read("/testdir", make([]byte, 2), 5, 0); status != -fuse.EISDIR {
		t.Errorf("Read returned %d bytes, want status %d", status, -fuse.EISDIR)
	}

	if status := fs.Read("/not_existing", make([]byte, 2), 5, 0); status != -fuse.ENOENT {
		t.Errorf("Read returned %d bytes, want status %d", status, -fuse.ENOENT)
	}
}

func TestWriteAt(t *testing.T) {
	initial := "hello"
	change := "XY"
	expected := "hXYlo"
	_, fs := createTestFs()
	if errc, _ := fs.Create("/file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700); errc != fuse.F_OK {
		t.Fatalf("Create result %d, want %d", errc, fuse.F_OK)
	}
	if errc := fs.Write("/file.txt", []byte(initial), 0, 0); errc != len(initial) {
		t.Fatalf("Write result %d, want %d", errc, len(initial))
	}
	if errc := fs.Write("/file.txt", []byte(change), 1, 0); errc != len(change) {
		t.Fatalf("Write result %d, want %d", errc, len(change))
	}
	buff := make([]byte, 100)
	read := fs.Read("/file.txt", buff, 0, 0)

	if read != len(expected) {
		t.Fatalf("Read returned %d, want %d", read, len(expected))
	}

	actual := string(buff[0:read])

	if actual != expected {
		t.Fatalf("Expected %s but got %s", expected, actual)
	}
}

func TestWriteAtToDir(t *testing.T) {
	data := "hello"
	store, fs := createTestFs()
	if err := store.HelperAddDir("/dir"); err != nil {
		t.Fatalf("CreateDir failed, %v", err)
	}

	errc := fs.Write("/dir", []byte(data), 0, 0)
	if errc != -fuse.EISDIR {
		t.Fatalf("Write result %d, want %d", errc, -fuse.EISDIR)
	}
}

func TestWriteWithNegOffset(t *testing.T) {
	data := "hello"
	_, fs := createTestFs()
	if errc, _ := fs.Create("/file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700); errc != fuse.F_OK {
		t.Fatalf("Create result %d, want %d", errc, fuse.F_OK)
	}

	errc := fs.Write("/file.txt", []byte(data), -1, 0)
	if errc != -fuse.EINVAL {
		t.Fatalf("Write result %d, want %d", errc, -fuse.EINVAL)
	}
}

func TestWriteWithEmptyWrite(t *testing.T) {

	_, fs := createTestFs()
	if errc, _ := fs.Create("/file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700); errc != fuse.F_OK {
		t.Fatalf("Create result %d, want %d", errc, fuse.F_OK)
	}

	n := fs.Write("/file.txt", []byte(""), 0, 0)
	if n != 0 {
		t.Fatalf("Write result %d, want %d", n, 0)
	}
}

func TestWriteToMissingFile(t *testing.T) {
	_, fs := createTestFs()

	errc := fs.Write("/file.txt", []byte(""), 0, 0)
	if errc != -fuse.ENOENT {
		t.Fatalf("Write result %d, want %d", errc, -fuse.ENOENT)
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

	if err := store.HelperAddFile("/file.txt", nil); err != nil {
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
	if err := store.HelperAddDir("/dir"); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

	result := fs.Read("/dir", make([]byte, 10), 0, 0)

	if result != -fuse.EISDIR {
		t.Errorf("Read returned %d, want %d", result, -fuse.EISDIR)
	}
}

func TestCreate(t *testing.T) {
	store, fs := createTestFs()
	if err := store.HelperAddDir("/dir"); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

	result, _ := fs.Create("/dir/file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700)
	if result != fuse.F_OK {
		t.Errorf("Create returned %d, want %d", result, fuse.F_OK)
	}

	meta, err := store.Stat("/dir/file.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if meta.Type != storage.NodeTypeFile {
		t.Fatalf("MetaData Type mismatch: got %s, want %s", meta.Type, storage.NodeTypeFile)
	}
	if meta.Size != 0 {
		t.Fatalf("MetaData Size mismatch: got %d, want %d", meta.Size, 0)
	}
	if meta.Mode != 0700 {
		t.Fatalf("MetaData Mode mismatch: got %#o, want %#o", meta.Mode, 0700)
	}

}

func TestCreateNoAccess(t *testing.T) {
	store, fs := createTestFs()
	if err := store.HelperAddDir("/dir"); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

	if err := store.HelperChangeMode("/dir", 0000); err != nil {
		t.Fatalf("ChangeMode failed: %v", err)
	}
	result, _ := fs.Create("/dir/file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700)
	if result != -fuse.EACCES {
		t.Errorf("Create returned %d, want %d", result, -fuse.EACCES)
	}
}

func TestCreateAlreadyExists(t *testing.T) {
	store, fs := createTestFs()
	if err := store.HelperAddDir("/dir"); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

	result, _ := fs.Create("/dir/file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700)
	if result != fuse.F_OK {
		t.Errorf("Create returned %d, want %d", result, fuse.F_OK)
	}

	result, _ = fs.Create("/dir/file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700)
	if result != -fuse.EEXIST {
		t.Errorf("Create returned %d, want %d", result, -fuse.EEXIST)
	}
}

func TestCreateParentIsFile(t *testing.T) {
	_, fs := createTestFs()

	result, _ := fs.Create("/file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700)
	if result != fuse.F_OK {
		t.Errorf("Create returned %d, want %d", result, fuse.F_OK)
	}

	result, _ = fs.Create("/file.txt/file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700)
	if result != -fuse.ENOTDIR {
		t.Errorf("Create returned %d, want %d", result, -fuse.ENOTDIR)
	}
}

func TestCreateMissingParentPath(t *testing.T) {
	_, fs := createTestFs()

	result, _ := fs.Create("/dir/file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700)
	if result != -fuse.ENOENT {
		t.Errorf("Create returned %d, want %d", result, -fuse.ENOENT)
	}
}

func TestCreateInvalidPath(t *testing.T) {
	_, fs := createTestFs()

	result, _ := fs.Create("//file.txt", fuse.O_WRONLY|fuse.O_CREAT, 0700)
	if result != -fuse.EINVAL {
		t.Errorf("Create returned %d, want %d", result, -fuse.EINVAL)
	}
}

func TestReadNegativeOffsetReturnsEINVAL(t *testing.T) {
	store, fs := createTestFs()
	if err := store.HelperAddFile("/file.txt", []byte("hello")); err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}

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
		{"already exists", storage.ErrAlreadyExists, -fuse.EEXIST},
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

func TestCheckAccess(t *testing.T) {
	tests := []struct {
		name string
		mode int
		mask uint32
		want bool
	}{
		{name: "existence check", mode: 0000, mask: fuse.F_OK, want: true},
		{name: "read allowed", mode: 0644, mask: fuse.R_OK, want: true},
		{name: "write allowed", mode: 0644, mask: fuse.W_OK, want: true},
		{name: "execute denied", mode: 0644, mask: fuse.X_OK, want: false},
		{name: "read and write allowed", mode: 0644, mask: fuse.R_OK | fuse.W_OK, want: true},
		{name: "read and execute denied", mode: 0644, mask: fuse.R_OK | fuse.X_OK, want: false},
		{name: "execute allowed", mode: 0755, mask: fuse.X_OK, want: true},
		{name: "no permissions", mode: 0000, mask: fuse.R_OK, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkAccess(tt.mode, tt.mask)
			if got != tt.want {
				t.Errorf("checkAccess(%#o, %d) = %t, want %t", tt.mode, tt.mask, got, tt.want)
			}
		})
	}
}

func TestOpenAccessMask(t *testing.T) {
	tests := []struct {
		name  string
		flags int
		want  uint32
	}{
		{name: "read only", flags: fuse.O_RDONLY, want: fuse.R_OK},
		{name: "write only", flags: fuse.O_WRONLY, want: fuse.W_OK},
		{name: "read write", flags: fuse.O_RDWR, want: fuse.R_OK | fuse.W_OK},
		{name: "additional flags", flags: fuse.O_WRONLY | fuse.O_CREAT, want: fuse.W_OK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := openAccessMask(tt.flags)
			if !ok {
				t.Errorf("openAccessMask(%d) returned not ok", tt.flags)
			}
			if got != tt.want {
				t.Errorf("openAccessMask(%d) = %d, want %d", tt.flags, got, tt.want)
			}
		})
	}
}

func TestOpenAccessMaskInvalid(t *testing.T) {
	flags := fuse.O_WRONLY | fuse.O_RDWR
	_, ok := openAccessMask(flags)
	if ok {
		t.Fatalf("openAccessMask(%d) returned ok for invalid flags", flags)
	}
}

func createTestFs() (*storage.InMemoryStorage, *FileSystem) {
	store := storage.NewInMemoryStorage()
	fs := NewFileSystem(store)
	return store, fs
}
