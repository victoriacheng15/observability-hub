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

	"gopkg.in/yaml.v3"
)

// --- Data Models ---

type Feature struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Icon        string `yaml:"icon"`
}

type Hero struct {
	Title    string `yaml:"title"`
	Subtitle string `yaml:"subtitle"`
	CtaText  string `yaml:"cta_text"`
	CtaLink  string `yaml:"cta_link"`
	Cta2Text string `yaml:"cta2_text"`
	Cta2Link string `yaml:"cta2_link"`
}

type Landing struct {
	PageTitle  string    `yaml:"page_title"`
	Hero       Hero      `yaml:"hero"`
	Philosophy []Feature `yaml:"philosophy"`
}

type Artifact struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type Event struct {
	Date             string     `yaml:"date"`
	Title            string     `yaml:"title"`
	Description      string     `yaml:"description"`
	DescriptionLines []string   `yaml:"-"`
	FormattedDate    string     `yaml:"-"`
	Artifacts        []Artifact `yaml:"artifacts"`
}

type Evolution struct {
	PageTitle string  `yaml:"page_title"`
	IntroText string  `yaml:"intro_text"`
	Timeline  []Event `yaml:"timeline"`
}

type DashboardItem struct {
	Caption   string `yaml:"caption"`
	ImagePath string `yaml:"image_path"`
}

type DashboardCategory struct {
	Name  string          `yaml:"name"`
	Items []DashboardItem `yaml:"items"`
}

type Dashboards struct {
	PageTitle  string              `yaml:"page_title"`
	Categories []DashboardCategory `yaml:"categories"`
}

type SiteData struct {
	Landing    Landing
	Evolution  Evolution
	Dashboards Dashboards
	Year       int
}

// --- Main Logic ---

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
	var data SiteData
	data.Year = time.Now().Year()

	configs := []struct {
		path string
		dest interface{}
	}{
		{"content/landing.yaml", &data.Landing},
		{"content/dashboards.yaml", &data.Dashboards},
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
	renderPage("index.html", "templates/index.html", &data)
	renderPage("evolution.html", "templates/evolution.html", &data)
	renderPage("dashboards.html", "templates/dashboards.html", &data)

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

func renderPage(outFile, tplFile string, data *SiteData) {
	// Always parse base.html + the specific page template
	tmpl, err := template.ParseFiles("templates/base.html", tplFile)
	if err != nil {
		log.Fatalf("Failed to parse templates for %s: %v", outFile, err)
	}

	f, err := os.Create(filepath.Join("dist", outFile))
	if err != nil {
		log.Fatalf("Failed to create output file %s: %v", outFile, err)
	}
	defer f.Close()

	// Execute "base" which should include the specific page content
	if err := tmpl.ExecuteTemplate(f, "base", data); err != nil {
		log.Fatalf("Failed to execute template %s: %v", outFile, err)
	}
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
