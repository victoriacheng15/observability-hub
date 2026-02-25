package schema

// Feature represents a single feature with a title, description, and icon.
type Feature struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Icon        string `yaml:"icon"`
}

// Hero represents the hero section of the landing page.
type Hero struct {
	Title    string `yaml:"title"`
	Subtitle string `yaml:"subtitle"`
	CtaText  string `yaml:"cta_text"`
	CtaLink  string `yaml:"cta_link"`
	Cta2Text string `yaml:"cta2_text"`
	Cta2Link string `yaml:"cta2_link"`
}

// Author represents the creator of the site.
type Author struct {
	Name     string `yaml:"name"`
	Github   string `yaml:"github"`
	Linkedin string `yaml:"linkedin"`
}

// Landing holds the data for the landing page.
type Landing struct {
	PageTitle       string    `yaml:"page_title"`
	SiteURL         string    `yaml:"site_url"`
	MetaDescription string    `yaml:"meta_description"`
	Keywords        string    `yaml:"keywords"`
	Author          Author    `yaml:"author"`
	Hero            Hero      `yaml:"hero"`
	Principles      []Feature `yaml:"principles"`
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
