package pathutil

import "testing"

func TestSplitParent(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantParent string
		wantName   string
		wantOK     bool
	}{
		{name: "root child", path: "/file.txt", wantParent: "/", wantName: "file.txt", wantOK: true},
		{name: "nested child", path: "/dir/file.txt", wantParent: "/dir", wantName: "file.txt", wantOK: true},
		{name: "empty", path: "", wantOK: false},
		{name: "relative", path: "dir/file.txt", wantOK: false},
		{name: "root", path: "/", wantOK: false},
		{name: "trailing slash", path: "/dir/", wantOK: false},
		{name: "empty component", path: "/dir//file.txt", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, name, ok := SplitParent(tt.path)
			if parent != tt.wantParent || name != tt.wantName || ok != tt.wantOK {
				t.Errorf(
					"SplitParent(%q) = (%q, %q, %t), want (%q, %q, %t)",
					tt.path,
					parent,
					name,
					ok,
					tt.wantParent,
					tt.wantName,
					tt.wantOK,
				)
			}
		})
	}
}
