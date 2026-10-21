package fusefs

import (
	"errors"

	"github.com/dwilkolek/rake-fs/internal/pathutil"
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
	case errors.Is(err, storage.ErrAlreadyExists):
		return -fuse.EEXIST
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
	node, err := fs.storage.Stat(path)
	if err != nil {
		return storageErrno(err)
	}

	if !checkAccess(node.Mode, mask) {
		return -fuse.EACCES
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

	mask, ok := openAccessMask(flags)
	if !ok || !checkAccess(meta.Mode, mask) {
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

func (fs *FileSystem) Create(path string, flags int, mode uint32) (int, uint64) {

	parentPath, _, ok := pathutil.SplitParent(path)
	if !ok {
		return -fuse.EINVAL, 0
	}

	meta, err := fs.storage.Stat(parentPath)
	if err != nil {
		return storageErrno(err), 0
	}
	if meta.Type != storage.NodeTypeDirectory {
		return -fuse.ENOTDIR, 0
	}
	if !checkAccess(meta.Mode, fuse.W_OK|fuse.X_OK) {
		return -fuse.EACCES, 0
	}

	err = fs.storage.Create(path, int(mode))
	if err != nil {
		return storageErrno(err), 0
	}

	return fuse.F_OK, 0
}

func (fs *FileSystem) Write(path string, buff []byte, ofst int64, fh uint64) int {
	n, err := fs.storage.WriteAt(path, buff, ofst)
	if n > 0 {
		return n
	}
	if err != nil {
		return storageErrno(err)
	}

	if len(buff) > 0 {
		return -fuse.EIO // avoid a zero-progress write
	}
	return n
}

// mode owner=110 group=100 other=100 = 644
// mask 100=r--, 110=rw-, 111=rwx
func checkAccess(mode int, mask uint32) bool {
	ownerPermissions := (mode >> 6) & 7

	//requested read && ownerPermissions has read = 0
	if mask&fuse.R_OK != 0 && ownerPermissions&4 == 0 {
		return false
	}

	if mask&fuse.W_OK != 0 && ownerPermissions&2 == 0 {
		return false
	}

	if mask&fuse.X_OK != 0 && ownerPermissions&1 == 0 {
		return false
	}

	return true
}

func openAccessMask(flags int) (uint32, bool) {
	switch flags & fuse.O_ACCMODE {
	case fuse.O_RDONLY:
		return fuse.R_OK, true
	case fuse.O_WRONLY:
		return fuse.W_OK, true
	case fuse.O_RDWR:
		return fuse.R_OK | fuse.W_OK, true
	default:
		return 0, false
	}
}
