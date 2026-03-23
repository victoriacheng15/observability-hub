package web

// Feature represents a single feature with a name, description, and icon.
type Feature struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Icon        string `yaml:"icon"`
}

// Header represents the header section of the landing page.
type Header struct {
	ProjectName string `yaml:"project_name"`
	SiteURL     string `yaml:"site_url"`
}

// SystemSpec represents the system specification for LLM discovery.
type SystemSpec struct {
	Objective           string `yaml:"objective"`
	Stack               string `yaml:"stack"`
	Pattern             string `yaml:"pattern"`
	EntryPoint          string `yaml:"entry_point"`
	PersistenceStrategy string `yaml:"persistence_strategy"`
	Observability       string `yaml:"observability"`
	MachineRegistry     string `yaml:"machine_registry"`
}

// Hero represents the hero section of the landing page.
type Hero struct {
	Headline         string `yaml:"headline"`
	SubHeadline      string `yaml:"sub_headline"`
	BriefDescription string `yaml:"brief_description"`
	CtaText          string `yaml:"cta_text"`
	CtaLink          string `yaml:"cta_link"`
	SecondaryCtaText string `yaml:"secondary_cta_text"`
	SecondaryCtaLink string `yaml:"secondary_cta_link"`
	TertiaryCtaText  string `yaml:"tertiary_cta_text"`
	TertiaryCtaLink  string `yaml:"tertiary_cta_link"`
}

// WhatIs represents the "What is" section.
type WhatIs struct {
	Title   string   `yaml:"title"`
	Content []string `yaml:"content"`
}

// KeyFeatures represents the key features section.
type KeyFeatures struct {
	Title    string    `yaml:"title"`
	Features []Feature `yaml:"features"`
}

// WhyItMatters represents the "Why it matters" section.
type WhyItMatters struct {
	Title  string   `yaml:"title"`
	Points []string `yaml:"points"`
}

// Footer represents the footer section of the landing page.
type Footer struct {
	Author       string `yaml:"author"`
	GithubLink   string `yaml:"github_link"`
	LinkedinLink string `yaml:"linkedin_link"`
}

// Landing holds the data for the landing page.
type Landing struct {
	Header       Header       `yaml:"header"`
	SystemSpec   SystemSpec   `yaml:"system_specification"`
	Hero         Hero         `yaml:"hero"`
	WhatIs       WhatIs       `yaml:"what_is_observability_hub"`
	KeyFeatures  KeyFeatures  `yaml:"key_features"`
	WhyItMatters WhyItMatters `yaml:"why_it_matters"`
	Footer       Footer       `yaml:"footer"`
}

// Artifact represents a link to an external artifact related to an event.
type Artifact struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// Event represents a single event in the evolution timeline.
type Event struct {
	Date             string     `yaml:"date"`
	Title            string     `yaml:"title"`
	Description      string     `yaml:"description"`
	DescriptionLines []string   `yaml:"-"` // This field is processed from Description
	FormattedDate    string     `yaml:"-"` // This field is processed from Date
	Artifacts        []Artifact `yaml:"artifacts"`
}

// Chapter represents a collection of events in a specific phase of the project.
type Chapter struct {
	Title  string  `yaml:"title"`
	Intro  string  `yaml:"intro"`
	Events []Event `yaml:"timeline"`
}

// Evolution holds the data for the evolution timeline page.
type Evolution struct {
	PageTitle string    `yaml:"page_title"`
	IntroText string    `yaml:"intro_text"`
	Chapters  []Chapter `yaml:"chapters"`
}

// SiteData is the top-level structure holding all data for the site.
type SiteData struct {
	Landing   Landing
	Evolution Evolution
	Year      int
}
