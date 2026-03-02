package generator

import (
	"os"
	"strings"
	"testing"

	"web/schema"
)

type MockFS struct {
	RemoveAllFn func(path string) error
	MkdirAllFn  func(path string, perm os.FileMode) error
	ReadDirFn   func(dirname string) ([]os.DirEntry, error)
	ReadFileFn  func(filename string) ([]byte, error)
	WriteFileFn func(filename string, data []byte, perm os.FileMode) error
}

func (m *MockFS) RemoveAll(path string) error                   { return m.RemoveAllFn(path) }
func (m *MockFS) MkdirAll(path string, perm os.FileMode) error  { return m.MkdirAllFn(path, perm) }
func (m *MockFS) ReadDir(dirname string) ([]os.DirEntry, error) { return m.ReadDirFn(dirname) }
func (m *MockFS) ReadFile(filename string) ([]byte, error)      { return m.ReadFileFn(filename) }
func (m *MockFS) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return m.WriteFileFn(filename, data, perm)
}

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() os.FileMode          { return 0 }
func (m *mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func TestLoadYaml(t *testing.T) {
	oldFS := fs
	defer func() { fs = oldFS }()

	tests := []struct {
		name    string
		wantErr bool
		mockFn  func() FileSystem
	}{
		{
			name:    "Success",
			wantErr: false,
			mockFn: func() FileSystem {
				return &MockFS{
					ReadFileFn: func(path string) ([]byte, error) { return []byte(`header: {project_name: T}`), nil },
				}
			},
		},
		{
			name:    "Read Error",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					ReadFileFn: func(path string) ([]byte, error) { return nil, os.ErrPermission },
				}
			},
		},
		{
			name:    "Decode Error",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					ReadFileFn: func(path string) ([]byte, error) { return []byte("invalid: yaml: :"), nil },
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs = tt.mockFn()
			var landing schema.Landing
			err := loadYaml("any.yaml", &landing)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadYaml() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuild(t *testing.T) {
	oldFS := fs
	defer func() { fs = oldFS }()

	tests := []struct {
		name    string
		wantErr bool
		mockFn  func() FileSystem
	}{
		{
			name:    "Full Success",
			wantErr: false,
			mockFn: func() FileSystem {
				return &MockFS{
					RemoveAllFn: func(p string) error { return nil },
					MkdirAllFn:  func(p string, perm os.FileMode) error { return nil },
					ReadDirFn:   func(p string) ([]os.DirEntry, error) { return nil, nil },
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.Contains(p, "base.html") {
							return []byte(`{{define "base"}}{{template "content" .}}{{end}}`), nil
						}
						if strings.Contains(p, ".yaml") {
							return []byte(`header: {project_name: T}`), nil
						}
						return []byte(`{{define "content"}}OK{{end}}`), nil
					},
					WriteFileFn: func(p string, d []byte, perm os.FileMode) error { return nil },
				}
			},
		},
		{
			name:    "MkdirAll Dist Failure",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					RemoveAllFn: func(p string) error { return nil },
					MkdirAllFn:  func(p string, perm os.FileMode) error { return os.ErrPermission },
				}
			},
		},
		{
			name:    "RemoveAll Failure",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{RemoveAllFn: func(p string) error { return os.ErrPermission }}
			},
		},
		{
			name:    "MkdirAll API Failure",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					RemoveAllFn: func(p string) error { return nil },
					MkdirAllFn: func(p string, perm os.FileMode) error {
						if strings.HasSuffix(p, "api") {
							return os.ErrPermission
						}
						return nil
					},
					ReadFileFn: func(p string) ([]byte, error) { return []byte(`header: {project_name: T}`), nil },
					ReadDirFn:  func(p string) ([]os.DirEntry, error) { return nil, nil },
				}
			},
		},
		{
			name:    "Full Build with Static and Evolution",
			wantErr: false,
			mockFn: func() FileSystem {
				return &MockFS{
					RemoveAllFn: func(p string) error { return nil },
					MkdirAllFn:  func(p string, perm os.FileMode) error { return nil },
					ReadDirFn: func(p string) ([]os.DirEntry, error) {
						if strings.HasSuffix(p, "static") {
							return []os.DirEntry{&mockDirEntry{name: "test.png"}}, nil
						}
						return nil, nil
					},
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.HasSuffix(p, "landing.yaml") {
							return []byte(`header: {project_name: T}`), nil
						}
						if strings.HasSuffix(p, "evolution.yaml") {
							return []byte(`
chapters:
  - title: C1
    events:
      - date: "2024-01-01"
        description: "- Point 1"
      - date: "bad-date"
        description: "Bad"
`), nil
						}
						if strings.Contains(p, "base.html") {
							return []byte(`{{define "base"}}{{template "content" .}}{{end}}`), nil
						}
						return []byte(`{{define "content"}}OK{{end}}`), nil
					},
					WriteFileFn: func(p string, d []byte, perm os.FileMode) error { return nil },
				}
			},
		},
		{
			name:    "Static ReadDir Failure",
			wantErr: false, // Build currently ignores ReadDir error
			mockFn: func() FileSystem {
				return &MockFS{
					RemoveAllFn: func(p string) error { return nil },
					MkdirAllFn:  func(p string, perm os.FileMode) error { return nil },
					ReadDirFn:   func(p string) ([]os.DirEntry, error) { return nil, os.ErrPermission },
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.Contains(p, "base.html") {
							return []byte(`{{define "base"}}{{template "content" .}}{{end}}`), nil
						}
						if strings.Contains(p, ".yaml") {
							return []byte(`header: {project_name: T}`), nil
						}
						return []byte(`{{define "content"}}OK{{end}}`), nil
					},
					WriteFileFn: func(p string, d []byte, perm os.FileMode) error { return nil },
				}
			},
		},
		{
			name:    "Static Read Failure",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					RemoveAllFn: func(p string) error { return nil },
					MkdirAllFn:  func(p string, perm os.FileMode) error { return nil },
					ReadDirFn: func(p string) ([]os.DirEntry, error) {
						if strings.HasSuffix(p, "static") {
							return []os.DirEntry{&mockDirEntry{name: "t.png"}}, nil
						}
						return nil, nil
					},
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.HasSuffix(p, "t.png") {
							return nil, os.ErrPermission
						}
						return []byte(`header: {project_name: T}`), nil
					},
				}
			},
		},
		{
			name:    "Static Write Failure",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					RemoveAllFn: func(p string) error { return nil },
					MkdirAllFn:  func(p string, perm os.FileMode) error { return nil },
					ReadDirFn: func(p string) ([]os.DirEntry, error) {
						if strings.HasSuffix(p, "static") {
							return []os.DirEntry{&mockDirEntry{name: "t.png"}}, nil
						}
						return nil, nil
					},
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.HasSuffix(p, "t.png") {
							return []byte("PNG"), nil
						}
						return []byte(`header: {project_name: T}`), nil
					},
					WriteFileFn: func(p string, d []byte, perm os.FileMode) error {
						if strings.HasSuffix(p, "t.png") {
							return os.ErrPermission
						}
						return nil
					},
				}
			},
		},
		{
			name:    "RenderPage Failure",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					RemoveAllFn: func(p string) error { return nil },
					MkdirAllFn:  func(p string, perm os.FileMode) error { return nil },
					ReadDirFn:   func(p string) ([]os.DirEntry, error) { return nil, nil },
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.Contains(p, "base.html") {
							return []byte(`{{define "base"}}{{bad}}`), nil // Cause parse error
						}
						return []byte(`header: {project_name: T}`), nil
					},
				}
			},
		},
		{
			name:    "RenderTemplate Failure",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					RemoveAllFn: func(p string) error { return nil },
					MkdirAllFn:  func(p string, perm os.FileMode) error { return nil },
					ReadDirFn:   func(p string) ([]os.DirEntry, error) { return nil, nil },
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.HasSuffix(p, "index.html") || strings.HasSuffix(p, "evolution.html") {
							return []byte(`{{define "content"}}OK{{end}}`), nil
						}
						if strings.HasSuffix(p, "base.html") {
							return []byte(`{{define "base"}}{{template "content" .}}{{end}}`), nil
						}
						if strings.Contains(p, "llms.txt") {
							return []byte(`{{.NonExistent}}`), nil // Cause template error
						}
						return []byte(`header: {project_name: T}`), nil
					},
					WriteFileFn: func(p string, d []byte, perm os.FileMode) error { return nil },
				}
			},
		},
		{
			name:    "GenerateRegistry Failure",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					RemoveAllFn: func(p string) error { return nil },
					MkdirAllFn:  func(p string, perm os.FileMode) error { return nil },
					ReadDirFn:   func(p string) ([]os.DirEntry, error) { return nil, nil },
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.HasSuffix(p, "base.html") {
							return []byte(`{{define "base"}}{{template "content" .}}{{end}}`), nil
						}
						return []byte(`header: {project_name: T}`), nil
					},
					WriteFileFn: func(p string, d []byte, perm os.FileMode) error {
						if strings.HasSuffix(p, "evolution-registry.json") {
							return os.ErrPermission
						}
						return nil
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs = tt.mockFn()
			err := Build("src", "dst")
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRenderPage(t *testing.T) {
	oldFS := fs
	defer func() { fs = oldFS }()

	tests := []struct {
		name    string
		wantErr bool
		mockFn  func() FileSystem
	}{
		{
			name:    "Success",
			wantErr: false,
			mockFn: func() FileSystem {
				return &MockFS{
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.Contains(p, "base.html") {
							return []byte(`{{define "base"}}{{template "content" .}}{{end}}`), nil
						}
						return []byte(`{{define "content"}}OK{{end}}`), nil
					},
					WriteFileFn: func(p string, d []byte, perm os.FileMode) error { return nil },
				}
			},
		},
		{
			name:    "Base Parse Error",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.Contains(p, "base.html") {
							return []byte(`{{define "base"}}{{bad}}`), nil
						}
						return []byte(`OK`), nil
					},
				}
			},
		},
		{
			name:    "Page Parse Error",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.Contains(p, "base.html") {
							return []byte(`{{define "base"}}{{template "content" .}}{{end}}`), nil
						}
						return []byte(`{{bad}}`), nil
					},
				}
			},
		},
		{
			name:    "Write Failure",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					ReadFileFn: func(p string) ([]byte, error) {
						if strings.Contains(p, "base.html") {
							return []byte(`{{define "base"}}{{template "content" .}}{{end}}`), nil
						}
						return []byte(`{{define "content"}}OK{{end}}`), nil
					},
					WriteFileFn: func(p string, d []byte, perm os.FileMode) error { return os.ErrPermission },
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs = tt.mockFn()
			err := renderPage("out.html", "base.html", "page.html", &schema.SiteData{})
			if (err != nil) != tt.wantErr {
				t.Errorf("renderPage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRenderTemplate(t *testing.T) {
	oldFS := fs
	defer func() { fs = oldFS }()

	tests := []struct {
		name    string
		wantErr bool
		mockFn  func() FileSystem
	}{
		{
			name:    "Success",
			wantErr: false,
			mockFn: func() FileSystem {
				return &MockFS{
					ReadFileFn:  func(p string) ([]byte, error) { return []byte("OK"), nil },
					WriteFileFn: func(p string, d []byte, perm os.FileMode) error { return nil },
				}
			},
		},
		{
			name:    "Parse Failure",
			wantErr: true,
			mockFn: func() FileSystem {
				return &MockFS{
					ReadFileFn: func(p string) ([]byte, error) { return []byte("{{bad"), nil },
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs = tt.mockFn()
			err := renderTemplate("out.txt", "tpl.txt", nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOSFileSystem(t *testing.T) {
	f := &OSFileSystem{}
	tmp, _ := os.MkdirTemp("", "os-fs-test")
	defer os.RemoveAll(tmp)

	if err := f.MkdirAll(tmp, 0755); err != nil {
		t.Errorf("MkdirAll failed: %v", err)
	}
	if err := f.WriteFile(tmp+"/test", []byte("OK"), 0644); err != nil {
		t.Errorf("WriteFile failed: %v", err)
	}
	if _, err := f.ReadFile(tmp + "/test"); err != nil {
		t.Errorf("ReadFile failed: %v", err)
	}
	if _, err := f.ReadDir(tmp); err != nil {
		t.Errorf("ReadDir failed: %v", err)
	}
	if err := f.RemoveAll(tmp); err != nil {
		t.Errorf("RemoveAll failed: %v", err)
	}
}

func TestGenerateRegistry_Error(t *testing.T) {
	oldFS := fs
	defer func() { fs = oldFS }()
	fs = &MockFS{
		WriteFileFn: func(p string, d []byte, perm os.FileMode) error { return os.ErrPermission },
	}
	if err := generateRegistry("any", nil); err == nil {
		t.Error("Expected error from generateRegistry, got nil")
	}
}
