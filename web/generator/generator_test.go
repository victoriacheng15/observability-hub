package generator

import (
	"os"
	"path/filepath"
	"testing"

	"web/schema"
)

func TestLoadYaml(t *testing.T) {
	t.Run("Load Landing", func(t *testing.T) {
		tmpContent := `
header:
  project_name: "Test Project"
  site_url: "https://example.com"
system_specification:
  objective: "Test Objective"
hero:
  headline: "Test Headline"
  sub_headline: "Test Subheadline"
  cta_text: "Click Me"
  cta_link: "/test.html"
what_is_observability_hub:
  title: "What is it"
  content: ["Point 1"]
key_features:
  title: "Features"
  features:
    - name: "Feat 1"
      description: "Desc 1"
      icon: "rocket"
why_it_matters:
  title: "Why"
  points: ["Point 1"]
footer:
  author: "Author"
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

		if landing.Header.ProjectName != "Test Project" {
			t.Errorf("Expected 'Test Project', got '%s'", landing.Header.ProjectName)
		}
		if len(landing.KeyFeatures.Features) != 1 {
			t.Errorf("Expected 1 features item, got %d", len(landing.KeyFeatures.Features))
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
	srcDir, err := os.MkdirTemp("", "web-build-src")
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
	landingYaml := []byte(`
header:
  project_name: Test
system_specification:
  objective: Test
hero:
  cta_link: "https://github.com/test"
what_is_observability_hub:
  title: Test
key_features:
  title: Test
why_it_matters:
  title: Test
footer:
  author: Test
`)
	if err := os.WriteFile(filepath.Join(contentDir, "landing.yaml"), landingYaml, 0644); err != nil {
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
	webTpl := []byte(`{{define "content"}}Page{{end}}`)
	if err := os.WriteFile(filepath.Join(tplDir, "index.html"), webTpl, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tplDir, "evolution.html"), webTpl, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tplDir, "llms.txt"), []byte("LLMS"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tplDir, "robots.txt"), []byte("Robots"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("Successful Build", func(t *testing.T) {
		dstDir, err := os.MkdirTemp("", "web-build-dst")
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
		if _, err := os.Stat(filepath.Join(dstDir, "llms.txt")); os.IsNotExist(err) {
			t.Error("llms.txt not created")
		}
		if _, err := os.Stat(filepath.Join(dstDir, "robots.txt")); os.IsNotExist(err) {
			t.Error("robots.txt not created")
		}
	})
}

func TestRenderPage(t *testing.T) {
	testCases := []struct {
		name            string
		baseTpl         string
		webTpl          string
		outFileName     string
		mockProjectName string
		expectedError   bool
		expectedContent string
	}{
		{
			name:            "Successful Render",
			baseTpl:         `{{define "base"}}<html><body>{{template "content" .}}</body></html>{{end}}`,
			webTpl:          `{{define "content"}}<h1>{{.Landing.Header.ProjectName}}</h1>{{end}}`,
			mockProjectName: "Test Render Page",
			expectedError:   false,
			expectedContent: "<html><body><h1>Test Render Page</h1></body></html>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpRoot, err := os.MkdirTemp("", "render-test-root-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpRoot)

			tmpTemplatesDir := filepath.Join(tmpRoot, "templates")
			if err := os.Mkdir(tmpTemplatesDir, 0755); err != nil {
				t.Fatal(err)
			}

			baseTplPath := filepath.Join(tmpTemplatesDir, "base.html")
			if err := os.WriteFile(baseTplPath, []byte(tc.baseTpl), 0644); err != nil {
				t.Fatal(err)
			}

			webTplFileName := "test_web.html"
			webTplPath := filepath.Join(tmpTemplatesDir, webTplFileName)
			if err := os.WriteFile(webTplPath, []byte(tc.webTpl), 0644); err != nil {
				t.Fatal(err)
			}

			mockData := &schema.SiteData{
				Landing: schema.Landing{
					Header: schema.Header{
						ProjectName: tc.mockProjectName,
					},
				},
			}

			originalWd, _ := os.Getwd()
			os.Chdir(tmpRoot)
			defer os.Chdir(originalWd)

			if err := os.Mkdir("dist", 0755); err != nil {
				t.Fatal(err)
			}

			outFile := filepath.Join("dist", "output.html")
			err = renderPage(outFile, baseTplPath, webTplPath, mockData)

			if tc.expectedError {
				if err == nil {
					t.Error("Expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("renderPage failed unexpectedly: %v", err)
				}
				gotContent, _ := os.ReadFile(outFile)
				if string(gotContent) != tc.expectedContent {
					t.Errorf("Expected '%s', got '%s'", tc.expectedContent, string(gotContent))
				}
			}
		})
	}
}

func TestRenderTemplate(t *testing.T) {
	t.Run("Successful RenderTemplate", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "render-template-test")
		defer os.RemoveAll(tmpDir)

		tplPath := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(tplPath, []byte("Hello {{.Name}}"), 0644)

		outPath := filepath.Join(tmpDir, "out.txt")
		data := struct{ Name string }{Name: "World"}

		if err := renderTemplate(outPath, tplPath, data); err != nil {
			t.Fatalf("renderTemplate failed: %v", err)
		}

		content, _ := os.ReadFile(outPath)
		if string(content) != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", string(content))
		}
	})
}
