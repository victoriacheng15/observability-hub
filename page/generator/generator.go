package generator

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"page/schema"

	"gopkg.in/yaml.v3"
)

// Internal variables for testing
var (
	removeAll = os.RemoveAll
	mkdirAll  = os.MkdirAll
)

// Build generates the static site from srcDir into dstDir.
func Build(srcDir, dstDir string) error {
	// 1. Cleanup dist
	if err := removeAll(dstDir); err != nil {
		return fmt.Errorf("failed to remove dist: %w", err)
	}
	if err := mkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist: %w", err)
	}

	// 2. Load Modular Data
	var data schema.SiteData
	data.Year = time.Now().Year()

	configs := []struct {
		path string
		dest interface{}
	}{
		{filepath.Join(srcDir, "content/landing.yaml"), &data.Landing},
		{filepath.Join(srcDir, "content/evolution.yaml"), &data.Evolution},
	}

	for _, cfg := range configs {
		if err := loadYaml(cfg.path, cfg.dest); err != nil {
			return fmt.Errorf("critical data failure loading %s: %w", cfg.path, err)
		}
	}

	// Process Descriptions and Dates for each Chapter
	for ci := range data.Evolution.Chapters {
		for ei := range data.Evolution.Chapters[ci].Events {
			event := &data.Evolution.Chapters[ci].Events[ei]

			// 1. Process description lines
			lines := strings.Split(event.Description, "\n")
			var cleanLines []string
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					cleanLines = append(cleanLines, strings.TrimPrefix(trimmed, "- "))
				}
			}
			event.DescriptionLines = cleanLines

			// 2. Process and format date
			t, err := time.Parse("2006-01-02", event.Date)
			if err != nil {
				log.Printf("Warning: failed to parse date %s: %v", event.Date, err)
				event.FormattedDate = event.Date
			} else {
				event.FormattedDate = t.Format("2006-01-02")
			}
		}
		// Reverse Events within each chapter (Newest First)
		slices.Reverse(data.Evolution.Chapters[ci].Events)
	}

	// Reverse Chapters (Newest First)
	slices.Reverse(data.Evolution.Chapters)

	// 4. Render Pages
	pages := []struct {
		out string
		tpl string
	}{
		{"index.html", "templates/index.html"},
		{"evolution.html", "templates/evolution.html"},
	}

	for _, p := range pages {
		outFile := filepath.Join(dstDir, p.out)
		tplFile := filepath.Join(srcDir, p.tpl)
		baseFile := filepath.Join(srcDir, "templates/base.html")
		if err := renderPage(outFile, baseFile, tplFile, &data); err != nil {
			return fmt.Errorf("failed to render %s: %w", p.out, err)
		}
	}

	return nil
}

func loadYaml(path string, out interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(out); err != nil {
		return fmt.Errorf("failed to decode %s: %w", path, err)
	}
	return nil
}

func renderPage(outFile, baseFile, tplFile string, data *schema.SiteData) error {
	// Add template functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}

	// Always parse base.html + the specific page template
	tmpl, err := template.New(filepath.Base(tplFile)).Funcs(funcMap).ParseFiles(baseFile, tplFile)
	if err != nil {
		return fmt.Errorf("failed to parse templates for %s: %w", outFile, err)
	}

	f, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outFile, err)
	}
	defer f.Close()

	// Execute "base" which should include the specific page content
	if err := tmpl.ExecuteTemplate(f, "base", data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", outFile, err)
	}

	return nil
}
