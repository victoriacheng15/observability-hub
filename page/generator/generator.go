package generator

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

	"page/schema"

	"gopkg.in/yaml.v3"
)

// Internal variables for testing
var (
	removeAll = os.RemoveAll
	mkdirAll  = os.MkdirAll
	readDir   = os.ReadDir
	readFile  = os.ReadFile
	writeFile = os.WriteFile
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

	// 3. Copy Static Assets
	staticDir := filepath.Join(srcDir, "static")
	if entries, err := readDir(staticDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				src := filepath.Join(staticDir, entry.Name())
				dst := filepath.Join(dstDir, entry.Name())
				content, err := readFile(src)
				if err != nil {
					return fmt.Errorf("failed to read static file %s: %w", entry.Name(), err)
				}
				if err := writeFile(dst, content, 0644); err != nil {
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

	// 5. Generate llms.txt
	if err := generateLLMS(filepath.Join(dstDir, "llms.txt"), &data.Landing); err != nil {
		return fmt.Errorf("failed to generate llms.txt: %w", err)
	}

	// 6. Generate evolution-registry.json
	apiDir := filepath.Join(dstDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return fmt.Errorf("failed to create api dir: %w", err)
	}
	if err := generateRegistry(filepath.Join(apiDir, "evolution-registry.json"), &data.Evolution); err != nil {
		return fmt.Errorf("failed to generate evolution-registry.json: %w", err)
	}

	return nil
}

func generateRegistry(path string, evolution *schema.Evolution) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(evolution)
}

func generateLLMS(path string, landing *schema.Landing) error {
	content := fmt.Sprintf(`# %s - System Specification

## Objective

%s

## Technical Stack

%s

## Core Architecture

- **Pattern**: %s
- **Entry Point**: %s
- **Persistence Strategy**: %s
- **Observability**: %s

## Discovery & Registry

- **Machine Registry**: %s

## Key 

- **GitHub Repository**: %s
`, landing.PageTitle, landing.Hero.Subtitle, landing.Spec.Stack, landing.Spec.Pattern, landing.Spec.EntryPoint, landing.Spec.PersistenceStrategy, landing.Spec.Observability, landing.Spec.MachineRegistry, landing.Hero.CtaLink)

	return os.WriteFile(path, []byte(content), 0644)
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
