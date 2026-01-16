package main

import (
	"os"
	"path/filepath"
	"testing"

	"page/schema"
)

func TestLoadYaml(t *testing.T) {
	t.Run("Load Landing", func(t *testing.T) {
		tmpContent := `
page_title: "Test Landing"
hero:
  title: "Test Hero"
  subtitle: "Test Subtitle"
  cta_text: "Click Me"
  cta_link: "/test.html"
  cta2_text: "Click Me 2"
  cta2_link: "/test2.html"
principles:
  - title: "Feat 1"
    description: "Desc 1"
    icon: "rocket"
`
		tmpFile, err := os.CreateTemp("", "landing-*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write([]byte(tmpContent)); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		var landing schema.Landing
		if err := loadYaml(tmpFile.Name(), &landing); err != nil {
			t.Fatalf("loadYaml failed for landing: %v", err)
		}

		if landing.PageTitle != "Test Landing" {
			t.Errorf("Expected 'Test Landing', got '%s'", landing.PageTitle)
		}
		if len(landing.Principles) != 1 {
			t.Errorf("Expected 1 principles item, got %d", len(landing.Principles))
		}
	})

	t.Run("Load Evolution", func(t *testing.T) {
		tmpContent := `
page_title: "Test Evolution"
intro_text: "Test Intro"
timeline:
  - date: "2024-01-01"
    title: "Event 1"
    description: "Desc 1"
`
		tmpFile, err := os.CreateTemp("", "evolution-*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write([]byte(tmpContent)); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		var evolution schema.Evolution
		if err := loadYaml(tmpFile.Name(), &evolution); err != nil {
			t.Fatalf("loadYaml failed for evolution: %v", err)
		}

		if evolution.PageTitle != "Test Evolution" {
			t.Errorf("Expected 'Test Evolution', got '%s'", evolution.PageTitle)
		}
		if len(evolution.Timeline) != 1 {
			t.Errorf("Expected 1 timeline event, got %d", len(evolution.Timeline))
		}
	})
}

func TestCopyFile(t *testing.T) {
	testCases := []struct {
		name        string
		srcContent  string
		setup       func(srcPath string)
		expectError bool
	}{
		{
			name:       "Successful Copy",
			srcContent: "Hello, Gopher!",
			setup: func(srcPath string) {
				if err := os.WriteFile(srcPath, []byte("Hello, Gopher!"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectError: false,
		},
		{
			name:        "Source Not Found",
			srcContent:  "",
			setup:       func(srcPath string) { os.Remove(srcPath) },
			expectError: true,
		},
		{
			name:       "Empty Source File",
			srcContent: "",
			setup: func(srcPath string) {
				if err := os.WriteFile(srcPath, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srcFile, err := os.CreateTemp("", "src-*.txt")
			if err != nil {
				t.Fatal(err)
			}
			srcPath := srcFile.Name()
			srcFile.Close() // Close immediately
			defer os.Remove(srcPath)

			dstFile, err := os.CreateTemp("", "dst-*.txt")
			if err != nil {
				t.Fatal(err)
			}
			dstPath := dstFile.Name()
			dstFile.Close() // Close immediately
			defer os.Remove(dstPath)

			tc.setup(srcPath) // Pass only srcPath to setup

			err = copyFile(srcPath, dstPath)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("copyFile failed: %v", err)
				}
				gotContent, readErr := os.ReadFile(dstPath)
				if readErr != nil {
					t.Fatalf("Failed to read destination file: %v", readErr)
				}
				if string(gotContent) != tc.srcContent {
					t.Errorf("Expected '%s', got '%s'", tc.srcContent, string(gotContent))
				}
			}
		})
	}
}

func TestCopyDir(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(srcDir string)
		verify      func(dstDir string)
		expectError bool
		isDir       bool
	}{
		{
			name: "Successful Directory Copy",
			setup: func(srcDir string) {
				if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644); err != nil {
					t.Fatal(err)
				}
				subDir := filepath.Join(srcDir, "sub-dir")
				if err := os.Mkdir(subDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			verify: func(dstDir string) {
				content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
				if err != nil {
					t.Fatalf("Failed to read dst/file1.txt: %v", err)
				}
				if string(content1) != "content1" {
					t.Errorf("Expected 'content1', got '%s'", string(content1))
				}

				content2, err := os.ReadFile(filepath.Join(dstDir, "sub-dir", "file2.txt"))
				if err != nil {
					t.Fatalf("Failed to read dst/sub-dir/file2.txt: %v", err)
				}
				if string(content2) != "content2" {
					t.Errorf("Expected 'content2', got '%s'", string(content2))
				}
			},
			expectError: false,
			isDir:       true,
		},
		{
			name:        "Source Not a Directory",
			setup:       func(srcDir string) {},
			verify:      func(dstDir string) {},
			expectError: true,
			isDir:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var srcPath string
			var err error
			if tc.isDir {
				srcPath, err = os.MkdirTemp("", "src-dir-")
				if err != nil {
					t.Fatal(err)
				}
				defer os.RemoveAll(srcPath)
			} else {
				srcFile, err := os.CreateTemp("", "src-file-")
				if err != nil {
					t.Fatal(err)
				}
				srcFile.Close()
				srcPath = srcFile.Name()
				defer os.Remove(srcPath)
			}

			dstDir, err := os.MkdirTemp("", "dst-dir-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dstDir)

			tc.setup(srcPath)

			err = copyDir(srcPath, dstDir)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("copyDir failed: %v", err)
				}
				tc.verify(dstDir)
			}
		})
	}
}

func TestRenderPage(t *testing.T) {
	testCases := []struct {
		name            string
		baseTpl         string
		pageTpl         string
		mockDataTitle   string
		expectedError   bool
		expectedContent string
	}{
		{
			name:            "Successful Render",
			baseTpl:         `{{define "base"}}<html><body>{{template "content" .}}</body></html>{{end}}`,
			pageTpl:         `{{define "content"}}<h1>{{.Landing.PageTitle}}</h1>{{end}}`,
			mockDataTitle:   "Test Render Page",
			expectedError:   false,
			expectedContent: "<html><body><h1>Test Render Page</h1></body></html>",
		},
		{
			name:            "Invalid Template Syntax",
			baseTpl:         `{{define "base"}}<html><body>{{template "content" .}}</body></html>{{end}}`,
			pageTpl:         `{{define "content"}}<h1>{{.Landing.PageTitle`, // Missing closing }} and {{end}}
			mockDataTitle:   "Test Render Page",
			expectedError:   true,
			expectedContent: "",
		},
		{
			name:            "Non-existent Template File",
			baseTpl:         `{{define "base"}}<html><body>{{template "content" .}}</body></html>{{end}}`,
			pageTpl:         "non_existent.html",
			mockDataTitle:   "Test Render Page",
			expectedError:   true,
			expectedContent: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for templates and output
			tmpRoot, err := os.MkdirTemp("", "render-test-root-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpRoot)

			// Create 'templates' subdirectory
			tmpTemplatesDir := filepath.Join(tmpRoot, "templates")
			if err := os.Mkdir(tmpTemplatesDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Create dummy base.html inside 'templates'
			if err := os.WriteFile(filepath.Join(tmpTemplatesDir, "base.html"), []byte(tc.baseTpl), 0644); err != nil {
				t.Fatal(err)
			}

			// Create dummy page template inside 'templates' if it's not a non-existent file test
			pageTplFileName := "test_page.html"
			if tc.pageTpl == "non_existent.html" {
				pageTplFileName = "non_existent.html"
			} else {
				if err := os.WriteFile(filepath.Join(tmpTemplatesDir, pageTplFileName), []byte(tc.pageTpl), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Mock SiteData
			mockData := &schema.SiteData{
				Landing: schema.Landing{
					PageTitle: tc.mockDataTitle,
				},
			}

			// Save current working directory and change to tmpRoot for rendering
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(tmpRoot); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(originalWd)

			// Create 'dist' subdirectory
			if err := os.Mkdir("dist", 0755); err != nil {
				t.Fatal(err)
			}

			outFile := "output.html"
			err = renderPage(outFile, "templates/"+pageTplFileName, mockData)

			if tc.expectedError {
				if err == nil {
					t.Error("Expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("renderPage failed unexpectedly: %v", err)
				}
				// Verify output file
				outputPath := filepath.Join("dist", outFile)
				gotContent, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file at %s: %v", outputPath, err)
				}
				if string(gotContent) != tc.expectedContent {
					t.Errorf("Expected '%s', got '%s'", tc.expectedContent, string(gotContent))
				}
			}
		})
	}
}
