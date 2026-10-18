package storage

import (
	"errors"
)

type NodeType string

var (
	ErrNotFound      = errors.New("not found")
	ErrNotDirectory  = errors.New("not a directory")
	ErrIsDirectory   = errors.New("is a directory")
	ErrInvalidPath   = errors.New("invalid path")
	ErrInvalidOffset = errors.New("invalid offset")
	ErrAlreadyExists = errors.New("already exists")
)

const (
	NodeTypeUnknown   NodeType = "unknown"
	NodeTypeFile      NodeType = "file"
	NodeTypeDirectory NodeType = "directory"
)

type Entry struct {
	Name string
}

type Storage interface {
	Stat(path string) (Metadata, error)
	List(path string) ([]Entry, error)
	ReadAt(path string, dst []byte, offset int64) (int, error)

	Create(path string, mode int) error
}

type Metadata struct {
	Type  NodeType
	Mode  int
	Size  int64
	NLink uint32
}
