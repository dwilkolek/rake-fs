package pathutil

import "strings"

// SplitParent splits an absolute filesystem path into its parent path and base
// name. It rejects the root path, relative paths, empty names, trailing slashes,
// and empty path components.
func SplitParent(path string) (parentPath, name string, ok bool) {
	if path == "" || path[0] != '/' || path == "/" || strings.HasSuffix(path, "/") || strings.Contains(path, "//") {
		return "", "", false
	}

	parentPath, name, found := strings.CutLast(path, "/")
	if !found || name == "" {
		return "", "", false
	}
	if parentPath == "" {
		parentPath = "/"
	}

	return parentPath, name, true
}
