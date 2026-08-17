package storage

import (
	"errors"
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

func TestStatDir(t *testing.T) {
	storage := setupTestStorage()
	meta, err := storage.Stat("/")

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
	if meta.Mode != 0555 {
		t.Fatalf("MetaData Mode mismatch: got %d, want %d", meta.Mode, 0555)
	}
}

func TestStatFile(t *testing.T) {
	storage := setupTestStorage()
	meta, err := storage.Stat("/swap")

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
	if meta.Mode != 0444 {
		t.Fatalf("MetaData Mode mismatch: got %d, want %d", meta.Mode, 0444)
	}
}

func TestStatNotExisting(t *testing.T) {
	storage := setupTestStorage()
	_, err := storage.Stat("/not-existing")

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Stat expected to error, got %q, want %q", err, ErrNotFound)
	}

}

func TestStatInvalidPath(t *testing.T) {
	storage := setupTestStorage()
	_, err := storage.Stat("/swap//file.txt")

	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Stat expected to error, got %q, want %q", err, ErrInvalidPath)
	}
}

func TestListValidPath(t *testing.T) {
	storage := setupTestStorage()
	entries, err := storage.List("/home")
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

func TestListOnFileh(t *testing.T) {
	storage := setupTestStorage()
	_, err := storage.List("/swap")

	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("List expected to error, got %q, want %q", err, ErrNotDirectory)
	}
}

func TestReatAt(t *testing.T) {
	storage := setupTestStorage()
	buff := make([]byte, 8)
	n, err := storage.ReadAt("/home/damian", buff, 6)

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

func TestReatAtEnd(t *testing.T) {
	storage := setupTestStorage()
	buff := make([]byte, 10)
	n, err := storage.ReadAt("/home/damian", buff, 15)

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

func setupTestStorage() *InMemoryStorage {
	storage := NewInMemoryStorage()
	storage.AddDir("/home")
	storage.AddFile("/home/julia", []byte("art is key"))
	storage.AddFile("/home/damian", []byte("smeet home, my home"))
	storage.AddDir("/home/others")

	storage.AddDir("/tmp")
	storage.AddFile("/swap", []byte("some random data"))
	return storage
}
