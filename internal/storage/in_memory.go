package storage

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"
)

const (
	readOnlyFileMode = 0444
	readOnlyDirMode  = 0555
)

type InMemoryStorage struct {
	mu   sync.RWMutex
	root *iNode
}

type iNode struct {
	nodeType NodeType
	data     []byte
	children map[string]*iNode
}

func (i *iNode) toMetadata() Metadata {

	var nlink int
	var mode int
	switch i.nodeType {
	case NodeTypeDirectory:
		nlink = 2
		for _, child := range i.children {
			if child.nodeType == NodeTypeDirectory {
				nlink++
			}
		}
		mode = readOnlyDirMode
	case NodeTypeFile:
		nlink = 1
		mode = readOnlyFileMode
	default:
		nlink = 0
		mode = 0000
	}

	return Metadata{
		Type:  i.nodeType,
		Mode:  mode,
		Size:  int64(len(i.data)),
		NLink: uint32(nlink),
	}
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		root: &iNode{
			nodeType: NodeTypeDirectory,
			children: map[string]*iNode{},
		},
	}
}

func (fs *InMemoryStorage) Stat(path string) (Metadata, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	node, err := fs.getNode(path)
	if err != nil {
		return Metadata{}, err
	}

	return node.toMetadata(), nil
}

func (fs *InMemoryStorage) List(path string) ([]Entry, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	node, err := fs.getNode(path)
	if err != nil {
		return nil, err
	}
	switch node.nodeType {
	case NodeTypeDirectory:
		entries := make([]Entry, 0, len(node.children))
		for name := range node.children {
			entries = append(entries, Entry{Name: name})
		}
		slices.SortStableFunc(entries, func(a Entry, b Entry) int {
			if node.children[a.Name].nodeType != node.children[b.Name].nodeType {
				if node.children[a.Name].nodeType == NodeTypeDirectory {
					return -1
				} else {
					return 1
				}
			}
			return cmp.Compare(a.Name, b.Name)
		})
		return entries, nil
	case NodeTypeFile:
		return nil, fmt.Errorf("%w: %q", ErrNotDirectory, path)
	default:
		return nil, fmt.Errorf("Unsupported node type for path: %s", path)
	}
}

func (fs *InMemoryStorage) ReadAt(path string, dst []byte, offset int64) (int, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	node, err := fs.getNode(path)
	if err != nil {
		return 0, err
	}
	if node.nodeType == NodeTypeDirectory {
		return 0, fmt.Errorf("%w: %q", ErrIsDirectory, path)
	}
	if offset < 0 {
		return 0, fmt.Errorf("%w: %d", ErrInvalidOffset, offset)
	}
	if offset >= int64(len(node.data)) {
		return 0, nil
	}

	n := copy(dst, node.data[offset:])
	return n, nil
}

func (fs *InMemoryStorage) mkdirs(path []string) (*iNode, error) {
	if path == nil {
		return nil, fmt.Errorf("%w: %q", ErrInvalidPath, path)
	}

	container := fs.root
	for _, key := range path {
		if container.nodeType != NodeTypeDirectory {
			return nil, fmt.Errorf("%w: %q", ErrNotDirectory, key)
		}
		childDir := container.children[key]
		if childDir == nil {
			childDir = &iNode{
				nodeType: NodeTypeDirectory,
				children: map[string]*iNode{},
			}
			container.children[key] = childDir
		}
		container = childDir
	}
	return container, nil
}

func (fs *InMemoryStorage) getNode(path string) (*iNode, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("%w: %q", ErrInvalidPath, path)
	}
	if path == "/" {
		return fs.root, nil
	}
	parts := strings.Split(path, "/")[1:]
	container := fs.root
	for _, key := range parts {
		if key == "" {
			return nil, fmt.Errorf("%w: %q", ErrInvalidPath, path)
		}
		if container.nodeType != NodeTypeDirectory {
			return nil, fmt.Errorf("%w: %q", ErrNotDirectory, key)
		}
		container = container.children[key]
		if container == nil {
			return nil, fmt.Errorf("%w: %q", ErrNotFound, path)
		}
	}
	return container, nil
}

// Helper methods
func (fs *InMemoryStorage) AddDir(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: %q", ErrInvalidPath, path)
	}
	_, err := fs.mkdirs(strings.Split(path, "/")[1:])
	return err
}

func (fs *InMemoryStorage) AddFile(path string, data []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: %q", ErrInvalidPath, path)
	}
	parts := strings.Split(path, "/")
	filename := parts[len(parts)-1]
	container, err := fs.mkdirs(parts[1 : len(parts)-1])
	if err != nil {
		return err
	}

	container.children[filename] = &iNode{
		nodeType: NodeTypeFile,
		data:     append([]byte(nil), data...),
	}
	return nil
}
