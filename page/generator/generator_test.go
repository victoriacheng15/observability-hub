package generator

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
  secondary_cta_text: "Click Me 2"
  secondary_cta_link: "/test2.html"
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
chapters:
  - title: "Chapter 1"
    intro: "Chapter Intro"
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
		if len(evolution.Chapters) != 1 {
			t.Errorf("Expected 1 chapter, got %d", len(evolution.Chapters))
		}
		if len(evolution.Chapters[0].Events) != 1 {
			t.Errorf("Expected 1 event in Chapter 0, got %d", len(evolution.Chapters[0].Events))
		}
	})
}

func TestBuild(t *testing.T) {
	// Setup temporary source directory
	srcDir, err := os.MkdirTemp("", "page-build-src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	// Setup content/ structure
	contentDir := filepath.Join(srcDir, "content")
	if err := os.Mkdir(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create dummy yaml files
	dummyYaml := []byte("page_title: Test\n")
	if err := os.WriteFile(filepath.Join(contentDir, "landing.yaml"), dummyYaml, 0644); err != nil {
		t.Fatal(err)
	}
	// Evolution needs special handling for chapters/events parsing
	evoYaml := []byte(`
page_title: Evolution
chapters:
  - title: C1
    events:
      - date: "2024-01-01"
        description: "- Point 1\n\n- Point 2"
      - date: "invalid-date"
        description: "Single Line"
`)
	if err := os.WriteFile(filepath.Join(contentDir, "evolution.yaml"), evoYaml, 0644); err != nil {
		t.Fatal(err)
	}

	// Setup templates/ structure
	tplDir := filepath.Join(srcDir, "templates")
	if err := os.Mkdir(tplDir, 0755); err != nil {
		t.Fatal(err)
	}
	baseTpl := []byte(`{{define "base"}}<html>{{template "content" .}}</html>{{end}}`)
	if err := os.WriteFile(filepath.Join(tplDir, "base.html"), baseTpl, 0644); err != nil {
		t.Fatal(err)
	}
	pageTpl := []byte(`{{define "content"}}Page{{end}}`)
	if err := os.WriteFile(filepath.Join(tplDir, "index.html"), pageTpl, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tplDir, "evolution.html"), pageTpl, 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("Successful Build", func(t *testing.T) {
		dstDir, err := os.MkdirTemp("", "page-build-dst")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dstDir)

		if err := Build(srcDir, dstDir); err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Verify output files exist
		if _, err := os.Stat(filepath.Join(dstDir, "index.html")); os.IsNotExist(err) {
			t.Error("index.html not created")
		}
		if _, err := os.Stat(filepath.Join(dstDir, "evolution.html")); os.IsNotExist(err) {
			t.Error("evolution.html not created")
		}
	})

	t.Run("Load Failure Last File", func(t *testing.T) {
		dstDir, err := os.MkdirTemp("", "page-build-fail-last")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dstDir)

		srcFail, _ := os.MkdirTemp("", "src-fail-last")
		defer os.RemoveAll(srcFail)

		// Setup content
		contentFail := filepath.Join(srcFail, "content")
		os.Mkdir(contentFail, 0755)
		os.WriteFile(filepath.Join(contentFail, "landing.yaml"), dummyYaml, 0644)
		// Broken evolution.yaml
		os.WriteFile(filepath.Join(contentFail, "evolution.yaml"), []byte("invalid: [ yaml"), 0644)

		err = Build(srcFail, dstDir)
		if err == nil {
			t.Error("Expected error due to broken evolution.yaml, got nil")
		}
	})

	t.Run("Missing Content File", func(t *testing.T) {
		dstDir, err := os.MkdirTemp("", "page-build-fail")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dstDir)

		// Create a src dir missing files
		emptySrc, _ := os.MkdirTemp("", "empty-src")
		defer os.RemoveAll(emptySrc)

		err = Build(emptySrc, dstDir)
		if err == nil {
			t.Error("Expected error due to missing content files, got nil")
		}
	})

	t.Run("Render Write Failure", func(t *testing.T) {
		// Let's test "Render Failure" by having a template that crashes during execution.
		// We modify the template in a separate temp src dir.

		brokenSrcDir, _ := os.MkdirTemp("", "broken-src")
		defer os.RemoveAll(brokenSrcDir)

		// Setup content
		os.Mkdir(filepath.Join(brokenSrcDir, "content"), 0755)
		os.WriteFile(filepath.Join(brokenSrcDir, "content/landing.yaml"), dummyYaml, 0644)
		os.WriteFile(filepath.Join(brokenSrcDir, "content/evolution.yaml"), evoYaml, 0644)

		// Create BROKEN templates
		brokenTplDir := filepath.Join(brokenSrcDir, "templates")
		os.Mkdir(brokenTplDir, 0755)
		os.WriteFile(filepath.Join(brokenTplDir, "base.html"), baseTpl, 0644)
		// Index template uses a function 'add' incorrectly?
		// funcMap has "add". {{ add 1 "string" }} will panic or error.
		badTpl := []byte(`{{define "content"}}{{ add 1 "string" }}{{end}}`)
		os.WriteFile(filepath.Join(brokenTplDir, "index.html"), badTpl, 0644)
		os.WriteFile(filepath.Join(brokenTplDir, "evolution.html"), pageTpl, 0644)

		dstDirRO, _ := os.MkdirTemp("", "dst-broken")
		defer os.RemoveAll(dstDirRO)

		err = Build(brokenSrcDir, dstDirRO)
		if err == nil {
			t.Error("Expected error due to bad template execution, got nil")
		} else {
			// Optional: check error message contains "executing"
		}
	})
	t.Run("Cleanup Failure", func(t *testing.T) {
		// Mock removeAll to fail
		originalRemoveAll := removeAll
		defer func() { removeAll = originalRemoveAll }()
		removeAll = func(path string) error {
			return os.ErrPermission
		}

		err := Build(srcDir, "dist-ignore")
		if err == nil {
			t.Error("Expected error due to removeAll failure, got nil")
		}
	})

	t.Run("Mkdir Failure", func(t *testing.T) {
		// Mock mkdirAll to fail
		originalMkdirAll := mkdirAll
		defer func() { mkdirAll = originalMkdirAll }()
		mkdirAll = func(path string, perm os.FileMode) error {
			return os.ErrPermission
		}

		err := Build(srcDir, "dist-ignore")
		if err == nil {
			t.Error("Expected error due to mkdirAll failure, got nil")
		}
	})
}

func TestRenderPage(t *testing.T) {
	testCases := []struct {
		name            string
		baseTpl         string
		pageTpl         string
		outFileName     string
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
			pageTpl:         `{{define "content"}}<h1>{{.Landing.PageTitle`,
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
		{
			name:            "Output Creation Failure",
			baseTpl:         `{{define "base"}}OK{{end}}`,
			pageTpl:         `{{define "content"}}OK{{end}}`,
			outFileName:     "missing-dir/output.html", // Parent dir doesn't exist
			mockDataTitle:   "Test",
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
			baseTplPath := filepath.Join(tmpTemplatesDir, "base.html")
			if err := os.WriteFile(baseTplPath, []byte(tc.baseTpl), 0644); err != nil {
				t.Fatal(err)
			}

			// Create dummy page template inside 'templates' if it's not a non-existent file test
			pageTplFileName := "test_page.html"
			pageTplPath := ""
			if tc.pageTpl == "non_existent.html" {
				pageTplPath = filepath.Join(tmpTemplatesDir, "non_existent.html")
			} else {
				pageTplPath = filepath.Join(tmpTemplatesDir, pageTplFileName)
				if err := os.WriteFile(pageTplPath, []byte(tc.pageTpl), 0644); err != nil {
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

			outName := "output.html"
			if tc.outFileName != "" {
				outName = tc.outFileName
			}
			outFile := filepath.Join("dist", outName)
			// Updated call signature
			err = renderPage(outFile, baseTplPath, pageTplPath, mockData)

			if tc.expectedError {
				if err == nil {
					t.Error("Expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("renderPage failed unexpectedly: %v", err)
				}
				// Verify output file
				gotContent, err := os.ReadFile(outFile)
				if err != nil {
					t.Fatalf("Failed to read output file at %s: %v", outFile, err)
				}
				if string(gotContent) != tc.expectedContent {
					t.Errorf("Expected '%s', got '%s'", tc.expectedContent, string(gotContent))
				}
			}
		})
	}
}
