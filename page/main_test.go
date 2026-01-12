package main

import (
	"os"
	"testing"
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
philosophy:
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

		var landing Landing
		if err := loadYaml(tmpFile.Name(), &landing); err != nil {
			t.Fatalf("loadYaml failed for landing: %v", err)
		}

		if landing.PageTitle != "Test Landing" {
			t.Errorf("Expected 'Test Landing', got '%s'", landing.PageTitle)
		}
		if len(landing.Philosophy) != 1 {
			t.Errorf("Expected 1 philosophy item, got %d", len(landing.Philosophy))
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

		var evolution Evolution
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
