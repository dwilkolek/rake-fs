package fusefs

import (
	"testing"

	"github.com/winfsp/cgofuse/fuse"
)

func TestGetattrRoot(t *testing.T) {
	fs := NewFileSystem()
	stat := &fuse.Stat_t{}

	result := fs.Getattr("/", stat, 0)

	if result != 0 {
		t.Fatalf("Getattr(/) returned %d, want 0", result)
	}
	if stat.Mode&fuse.S_IFMT != fuse.S_IFDIR {
		t.Errorf("root is not a directory: mode = %#o", stat.Mode)
	}
	if stat.Mode&0755 != 0755 {
		t.Errorf("permissions = %#o, want 0755", stat.Mode&0755)
	}
	if stat.Nlink != 2 {
		t.Errorf("Nlink = %d, want 2", stat.Nlink)
	}
}

func TestGetattrMissingPath(t *testing.T) {
	fs := NewFileSystem()
	stat := &fuse.Stat_t{}

	result := fs.Getattr("/missing", stat, 0)

	if result != -fuse.ENOENT {
		t.Errorf("Getattr(/missing) returned %d, want %d",
			result, -fuse.ENOENT)
	}
}

func TestReaddirRoot(t *testing.T) {
	fs := NewFileSystem()
	var names []string

	result := fs.Readdir(
		"/",
		func(name string, stat *fuse.Stat_t, offset int64) bool {
			names = append(names, name)
			return false
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
	expectedContent := "This is a test file."
	fs := NewFileSystem()
	fs.files["/testfile.txt"] = &FSNode{
		isDir: false,
		data:  []byte(expectedContent),
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
	fs := NewFileSystem()
	fs.files["/testfile.txt"] = &FSNode{
		isDir: false,
		data:  []byte("This is a test file."),
	}

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
	fs := NewFileSystem()
	fs.files["/dir"] = &FSNode{
		isDir: true,
	}

	if errc, _ := fs.Open("/test.txt", fuse.O_RDONLY); errc != -fuse.ENOENT {
		t.Errorf("Open returned %d, want %d", errc, -fuse.ENOENT)
	}

	if errc, _ := fs.Open("/dir", fuse.O_RDONLY); errc != -fuse.ENOENT {
		t.Errorf("Open returned %d, want %d", errc, -fuse.ENOENT)
	}

}

func TestReadFile(t *testing.T) {
	expectedContent := "is"
	fs := NewFileSystem()
	fs.files["/testfile.txt"] = &FSNode{
		isDir: false,
		data:  []byte("This is a test file."),
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
	fs := NewFileSystem()
	fs.files["/testdir"] = &FSNode{
		isDir: true,
	}

	if status := fs.Read("/testdir", make([]byte, 2), 5, 0); status != -fuse.ENOENT {
		t.Errorf("Read returned %d bytes, want %d", status, -fuse.ENOENT)
	}

	if status := fs.Read("/not_existing", make([]byte, 2), 5, 0); status != -fuse.ENOENT {
		t.Errorf("Read returned %d bytes, want %d", status, -fuse.ENOENT)
	}
}
