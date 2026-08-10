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
