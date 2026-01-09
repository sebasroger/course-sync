package domain

type UnifiedCourse struct {
	Source        string // "udemy" | "pluralsight"
	SourceID      string
	Title         string
	Description   string
	CourseURL     string
	Language      string
	Category      string
	Difficulty    string
	DurationHours float64
	Status        string // "active" | "inactive"
	PublishedDate string // ISO string if available
	ImageURL      string
	Skills        []string
}
