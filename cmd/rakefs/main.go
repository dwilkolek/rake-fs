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

	initTestData(store)
	defer host.Unmount()
	host.Mount(mountpoint, []string{"-d"})
}

func initTestData(fs *storage.InMemoryStorage) {
	fs.AddFile("/hello.txt", []byte("Hello, World!\nthis is a test file for the FUSE filesystem.\n Enjoy!"))
	fs.AddDir("/testdir")
}
