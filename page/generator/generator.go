package generator

import (
	"fmt"
	"html/template"
	"io"
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

	// 2. Copy Assets
	assetsSrc := filepath.Join(srcDir, "assets")
	assetsDst := filepath.Join(dstDir, "assets")
	if err := copyDir(assetsSrc, assetsDst); err != nil {
		// Log warning but continue, as assets might not exist in all test cases
		log.Printf("Warning copying assets: %v", err)
	}

	// 3. Load Modular Data
	var data schema.SiteData
	data.Year = time.Now().Year()

	configs := []struct {
		path string
		dest interface{}
	}{
		{filepath.Join(srcDir, "content/landing.yaml"), &data.Landing},
		{filepath.Join(srcDir, "content/snapshots.yaml"), &data.Snapshots},
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
				event.FormattedDate = t.Format("Jan 02, 2006")
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
		{"snapshots.html", "templates/snapshots.html"},
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

// copyDir recursively copies a directory tree, attempting to preserve permissions.
func copyDir(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dst, si.Mode())
		if err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Copy file
			if err = copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}
