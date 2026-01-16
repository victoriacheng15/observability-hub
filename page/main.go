package main

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

func main() {
	// 1. Cleanup dist
	if err := os.RemoveAll("dist"); err != nil {
		log.Fatalf("Failed to remove dist: %v", err)
	}
	if err := os.MkdirAll("dist", 0755); err != nil {
		log.Fatalf("Failed to create dist: %v", err)
	}

	// 2. Copy Assets
	if err := copyDir("assets", "dist/assets"); err != nil {
		log.Printf("Warning copying assets: %v", err)
	}

	// 3. Load Modular Data
	var data schema.SiteData
	data.Year = time.Now().Year()

	configs := []struct {
		path string
		dest interface{}
	}{
		{"content/landing.yaml", &data.Landing},
		{"content/snapshots.yaml", &data.Snapshots},
		{"content/evolution.yaml", &data.Evolution},
	}

	for _, cfg := range configs {
		if err := loadYaml(cfg.path, cfg.dest); err != nil {
			log.Fatalf("Critical data failure loading %s: %v", cfg.path, err)
		}
	}

	// Process Descriptions and Dates
	for i := range data.Evolution.Timeline {
		// 1. Process description lines
		lines := strings.Split(data.Evolution.Timeline[i].Description, "\n")
		var cleanLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				cleanLines = append(cleanLines, strings.TrimPrefix(trimmed, "- "))
			}
		}
		data.Evolution.Timeline[i].DescriptionLines = cleanLines

		// 2. Process and format date
		t, err := time.Parse("2006-01-02", data.Evolution.Timeline[i].Date)
		if err != nil {
			log.Printf("Warning: failed to parse date %s: %v", data.Evolution.Timeline[i].Date, err)
			data.Evolution.Timeline[i].FormattedDate = data.Evolution.Timeline[i].Date
		} else {
			data.Evolution.Timeline[i].FormattedDate = t.Format("Jan 02, 2006")
		}
	}

	// Reverse Timeline (Newest First)
	slices.Reverse(data.Evolution.Timeline)

	// 4. Render Pages
	if err := renderPage("index.html", "templates/index.html", &data); err != nil {
		log.Fatalf("Failed to render index.html: %v", err)
	}
	if err := renderPage("evolution.html", "templates/evolution.html", &data); err != nil {
		log.Fatalf("Failed to render evolution.html: %v", err)
	}
	if err := renderPage("snapshots.html", "templates/snapshots.html", &data); err != nil {
		log.Fatalf("Failed to render snapshots.html: %v", err)
	}

	fmt.Println("Site generated successfully in dist/")
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

func renderPage(outFile, tplFile string, data *schema.SiteData) error {
	// Always parse base.html + the specific page template
	tmpl, err := template.ParseFiles("templates/base.html", tplFile)
	if err != nil {
		return fmt.Errorf("failed to parse templates for %s: %w", outFile, err)
	}

	f, err := os.Create(filepath.Join("dist", outFile))
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
