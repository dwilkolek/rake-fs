package main

import (
	"log"
	"os"

	"github.com/dwilkolek/rake-fs/internal/fusefs"
	"github.com/dwilkolek/rake-fs/internal/storage"
	"github.com/winfsp/cgofuse/fuse"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: rakefs <mountpoint>")
	}
	mountpoint := os.Args[1]
	log.Printf("Mounting at %s", mountpoint)
	store := storage.NewInMemoryStorage()
	myfs := fusefs.NewFileSystem(store)
	host := fuse.NewFileSystemHost(myfs)

	err := initTestData(store)
	if err != nil {
		log.Fatalf("Failed init test data: %v", err)
	}
	defer host.Unmount()
	if !host.Mount(mountpoint, []string{"-d"}) {
		log.Fatalf("Failed to mount: %s", mountpoint)
	}
}

func initTestData(fs *storage.InMemoryStorage) error {
	if err := fs.HelperAddFile("/hello.txt", []byte("Hello, World!\nthis is a test file for the FUSE filesystem.\n Enjoy!")); err != nil {
		return err
	}
	if err := fs.HelperAddDir("/testdir"); err != nil {
		return err
	}
	return nil
}
