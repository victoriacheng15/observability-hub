package schema

// SnapshotItem represents a single snapshot image with its caption and path.
type SnapshotItem struct {
	Caption   string `yaml:"caption"`
	ImagePath string `yaml:"image_path"`
}

// SnapshotMonth represents a monthly collection of snapshots for a specific category.
type SnapshotMonth struct {
	Period string         `yaml:"period"`
	Items  []SnapshotItem `yaml:"items"`
}

// SnapshotCategory represents a high-level grouping of snapshots, like "Reading Analytics".
type SnapshotCategory struct {
	Name    string          `yaml:"name"`
	History []SnapshotMonth `yaml:"history"`
}

// Snapshots holds the overall structure for the snapshots page, including metadata and categories.
type Snapshots struct {
	PageTitle    string             `yaml:"page_title"`
	LastUpdated  string             `yaml:"last_updated"`
	NextUpdate   string             `yaml:"next_update"`
	ReportWindow string             `yaml:"report_window"`
	Categories   []SnapshotCategory `yaml:"categories"`
}

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

// Landing holds the data for the landing page.
type Landing struct {
	PageTitle  string    `yaml:"page_title"`
	Hero       Hero      `yaml:"hero"`
	Principles []Feature `yaml:"principles"`
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

// Evolution holds the data for the evolution timeline page.
type Evolution struct {
	PageTitle string  `yaml:"page_title"`
	IntroText string  `yaml:"intro_text"`
	Timeline  []Event `yaml:"timeline"`
}

// SiteData is the top-level structure holding all data for the site.
type SiteData struct {
	Landing   Landing
	Evolution Evolution
	Snapshots Snapshots
	Year      int
}
