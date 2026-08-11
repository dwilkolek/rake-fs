package fusefs

import (
	"strings"

	"github.com/winfsp/cgofuse/fuse"
)

type FileSystem struct {
	fuse.FileSystemBase
	files map[string]*FSNode
}

type FSNode struct {
	isDir bool
	data  []byte
}

func NewFileSystem() *FileSystem {
	return &FileSystem{
		files: map[string]*FSNode{
			"/": {
				isDir: true,
			},
		},
	}
}

func (fs *FileSystem) AddFile(path string, data []byte) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	fs.files[path] = &FSNode{
		isDir: false,
		data:  data,
	}
}

func (fs *FileSystem) AddDir(path string) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	fs.files[path] = &FSNode{
		isDir: true,
	}
}

func (fs *FileSystem) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	node := fs.files[path]
	if node == nil {
		return -fuse.ENOENT
	}
	if node.isDir {
		stat.Mode = fuse.S_IFDIR | 0755
		stat.Nlink = 2 // root directory has 2 links: itself and its parent
	} else {
		stat.Mode = fuse.S_IFREG | 0444 //Read only file
		stat.Nlink = 1
		stat.Size = int64(len(node.data))
	}
	return fuse.F_OK
}

func (fs *FileSystem) Access(path string, mask uint32) int {
	if fs.files[path] == nil {
		return -fuse.ENOENT
	}

	return fuse.F_OK
}

func (fs *FileSystem) Readdir(path string, fill func(name string, stat *fuse.Stat_t, ofst int64) bool, ofst int64, fh uint64) int {
	node := fs.files[path]
	if node == nil || !node.isDir {
		return -fuse.ENOENT
	}
	fill(".", nil, 0)
	fill("..", nil, 0)
	for name := range fs.files {
		if name == "/" {
			continue
		}
		fill(strings.TrimPrefix(name, "/"), nil, 0)
	}

	return fuse.F_OK
}

func (fs *FileSystem) Open(path string, flags int) (errc int, fh uint64) {
	node := fs.files[path]
	if node == nil || node.isDir {
		return -fuse.ENOENT, 0
	}

	if flags&fuse.O_ACCMODE != fuse.O_RDONLY {
		return -fuse.EACCES, 0
	}

	return fuse.F_OK, 0
}

func (fs *FileSystem) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {
	node := fs.files[path]
	if node == nil || node.isDir {
		return -fuse.ENOENT
	}

	if ofst >= int64(len(node.data)) {
		return 0
	}

	n = copy(buff, node.data[ofst:])
	return n
}
