package storage

import (
	"testing"
)

func TestAddDirs(t *testing.T) {
	storage := NewInMemoryStorage()
	err := storage.AddDir("/dir1/dir2/dir3")
	if err != nil {
		t.Fatalf("AddDir failed: %v", err)
	}
	dir1 := storage.root.children["dir1"]
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

func TestAddFiles(t *testing.T) {
	storage := NewInMemoryStorage()
	err := storage.AddFile("/dir1/dir2/file.txt", []byte("Hello, World!"))
	if err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	dir1 := storage.root.children["dir1"]
	if dir1 == nil || dir1.nodeType != NodeTypeDirectory {
		t.Fatalf("FindNode returned nil or a file for existing directory")
	}
	dir2 := dir1.children["dir2"]
	if dir2 == nil || dir2.nodeType != NodeTypeDirectory {
		t.Fatalf("FindNode returned nil or a file for existing directory")
	}
	file := dir2.children["file.txt"]

	if file == nil || file.nodeType != NodeTypeFile {
		t.Fatalf("FindNode returned nil or a directory for existing file")
	}
	if string(file.data) != "Hello, World!" {
		t.Fatalf("File data mismatch: got %q, want %q", string(file.data), "Hello, World!")
	}
}
