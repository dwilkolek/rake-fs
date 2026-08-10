package main

import (
	"log"
	"os"

	"github.com/dwilkolek/rake-fs/internal/fusefs"
	"github.com/winfsp/cgofuse/fuse"
)

func main() {
	mountpoint := os.Args[1]
	log.Printf("Mounting at %s", mountpoint)
	myfs := fusefs.NewFileSystem()
	host := fuse.NewFileSystemHost(myfs)
	host.Mount(mountpoint, []string{"-d"})

	defer host.Unmount()
}
