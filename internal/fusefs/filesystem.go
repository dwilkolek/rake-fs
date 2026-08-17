package fusefs

import (
	"errors"

	"github.com/dwilkolek/rake-fs/internal/storage"
	"github.com/winfsp/cgofuse/fuse"
)

type FileSystem struct {
	fuse.FileSystemBase
	storage storage.Storage
}

func storageErrno(err error) int {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		return -fuse.ENOENT
	case errors.Is(err, storage.ErrNotDirectory):
		return -fuse.ENOTDIR
	case errors.Is(err, storage.ErrIsDirectory):
		return -fuse.EISDIR
	case errors.Is(err, storage.ErrInvalidPath),
		errors.Is(err, storage.ErrInvalidOffset):
		return -fuse.EINVAL
	default:
		return -fuse.EIO
	}
}

func NewFileSystem(storage storage.Storage) *FileSystem {
	return &FileSystem{
		storage: storage,
	}
}

func (fs *FileSystem) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	meta, err := fs.storage.Stat(path)
	if err != nil {
		return storageErrno(err)
	}

	switch meta.Type {
	case storage.NodeTypeDirectory:
		stat.Mode = uint32(fuse.S_IFDIR | meta.Mode)
		stat.Nlink = meta.NLink
	case storage.NodeTypeFile:
		stat.Mode = uint32(fuse.S_IFREG | meta.Mode)
		stat.Size = meta.Size
		stat.Nlink = meta.NLink
	default:
		return -fuse.EIO
	}

	return fuse.F_OK
}

func (fs *FileSystem) Access(path string, mask uint32) int {
	if _, err := fs.storage.Stat(path); err != nil {
		return storageErrno(err)
	}

	return fuse.F_OK
}

func (fs *FileSystem) Readdir(path string, fill func(name string, stat *fuse.Stat_t, ofst int64) bool, ofst int64, fh uint64) int {
	entries, err := fs.storage.List(path)
	if err != nil {
		return storageErrno(err)
	}

	if !fill(".", nil, 0) {
		return fuse.F_OK
	}
	if !fill("..", nil, 0) {
		return fuse.F_OK
	}
	for _, entry := range entries {
		if !fill(entry.Name, nil, 0) {
			break
		}
	}

	return fuse.F_OK
}

func (fs *FileSystem) Open(path string, flags int) (errc int, fh uint64) {
	meta, err := fs.storage.Stat(path)
	if err != nil {
		return storageErrno(err), 0
	}
	if meta.Type == storage.NodeTypeDirectory {
		return -fuse.EISDIR, 0
	}
	if flags&fuse.O_ACCMODE != fuse.O_RDONLY || meta.Mode&0444 == 0 {
		return -fuse.EACCES, 0
	}

	return fuse.F_OK, 0
}

func (fs *FileSystem) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {
	if ofst < 0 {
		return -fuse.EINVAL
	}

	n, err := fs.storage.ReadAt(path, buff, ofst)

	if err != nil {
		return storageErrno(err)
	}

	return n
}
