package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FileSystem defines the interface for file operations to allow mocking.
type FileSystem interface {
	RemoveAll(path string) error
	MkdirAll(path string, perm os.FileMode) error
	ReadDir(dirname string) ([]os.DirEntry, error)
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

// OSFileSystem implements FileSystem using the standard os package.
type OSFileSystem struct{}

func (f *OSFileSystem) RemoveAll(path string) error                   { return os.RemoveAll(path) }
func (f *OSFileSystem) MkdirAll(path string, perm os.FileMode) error  { return os.MkdirAll(path, perm) }
func (f *OSFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) { return os.ReadDir(dirname) }
func (f *OSFileSystem) ReadFile(filename string) ([]byte, error)      { return os.ReadFile(filename) }
func (f *OSFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

// Global FS used by the generator, can be swapped in tests.
var fs FileSystem = &OSFileSystem{}

// Build generates the static site from srcDir into dstDir.
func Build(srcDir, dstDir string) error {
	// 1. Cleanup dist
	if err := fs.RemoveAll(dstDir); err != nil {
		return fmt.Errorf("failed to remove dist: %w", err)
	}
	if err := fs.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist: %w", err)
	}

	// 2. Load Modular Data
	var data SiteData
	data.Year = time.Now().Year()

	configs := []struct {
		path string
		dest interface{}
	}{
		{filepath.Join(srcDir, "templates/content/landing.yaml"), &data.Landing},
		{filepath.Join(srcDir, "templates/content/evolution.yaml"), &data.Evolution},
	}

	for _, cfg := range configs {
		if err := loadYaml(cfg.path, cfg.dest); err != nil {
			return fmt.Errorf("critical data failure loading %s: %w", cfg.path, err)
		}
	}

	// 3. Copy Static Assets
	staticDir := filepath.Join(srcDir, "static")
	if entries, err := fs.ReadDir(staticDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				src := filepath.Join(staticDir, entry.Name())
				dst := filepath.Join(dstDir, entry.Name())
				content, err := fs.ReadFile(src)
				if err != nil {
					return fmt.Errorf("failed to read static file %s: %w", entry.Name(), err)
				}
				if err := fs.WriteFile(dst, content, 0644); err != nil {
					return fmt.Errorf("failed to write static file %s: %w", entry.Name(), err)
				}
			}
		}
	}

	// 4. Process Descriptions and Dates for each Chapter
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
		slices.Reverse(data.Evolution.Chapters[ci].Events)
	}
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

	// 5. Generate Other Templates
	templates := []struct {
		out string
		tpl string
	}{
		{"llms.txt", "templates/llms.txt"},
		{"robots.txt", "templates/robots.txt"},
	}

	for _, t := range templates {
		if err := renderTemplate(filepath.Join(dstDir, t.out), filepath.Join(srcDir, t.tpl), &data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", t.out, err)
		}
	}

	// 6. Generate evolution-registry.json
	apiDir := filepath.Join(dstDir, "api")
	if err := fs.MkdirAll(apiDir, 0755); err != nil {
		return fmt.Errorf("failed to create api dir: %w", err)
	}
	if err := generateRegistry(filepath.Join(apiDir, "evolution-registry.json"), &data.Evolution); err != nil {
		return fmt.Errorf("failed to generate evolution-registry.json: %w", err)
	}

	return nil
}

func generateRegistry(path string, evolution *Evolution) error {
	data, err := json.Marshal(evolution)
	if err != nil {
		return err
	}
	return fs.WriteFile(path, data, 0644)
}

func renderTemplate(path string, tplPath string, data interface{}) error {
	tplContent, err := fs.ReadFile(tplPath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tplPath, err)
	}

	tmpl, err := template.New(filepath.Base(tplPath)).Parse(string(tplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var out strings.Builder
	if err := tmpl.Execute(&out, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", path, err)
	}

	return fs.WriteFile(path, []byte(out.String()), 0644)
}

func loadYaml(path string, out interface{}) error {
	content, err := fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	if err := yaml.Unmarshal(content, out); err != nil {
		return fmt.Errorf("failed to decode %s: %w", path, err)
	}
	return nil
}

func renderPage(outFile, baseFile, tplFile string, data *SiteData) error {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}

	baseContent, err := fs.ReadFile(baseFile)
	if err != nil {
		return fmt.Errorf("failed to read base template: %w", err)
	}

	tplContent, err := fs.ReadFile(tplFile)
	if err != nil {
		return fmt.Errorf("failed to read page template: %w", err)
	}

	tmpl, err := template.New("page").Funcs(funcMap).Parse(string(baseContent))
	if err != nil {
		return fmt.Errorf("failed to parse base template: %w", err)
	}
	tmpl, err = tmpl.Parse(string(tplContent))
	if err != nil {
		return fmt.Errorf("failed to parse page template: %w", err)
	}

	var out strings.Builder
	if err := tmpl.ExecuteTemplate(&out, "base", data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", outFile, err)
	}

	return fs.WriteFile(outFile, []byte(out.String()), 0644)
}
