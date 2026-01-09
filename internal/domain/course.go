package domain

// UnifiedCourse is the canonical representation of a course inside this service.
// All providers should map into this model, and all destinations (Eightfold, etc.)
// should map from this model.
type UnifiedCourse struct {
	Source      string  // "pluralsight", "udemy", etc.
	SourceID    string  // provider course id
	Title       string
	Description string
	CourseURL   string
	Language    string

	Category   string
	Difficulty string
	DurationHours float64

	Status string // "active", "inactive", etc.
	PublishedDate string // ISO string if available
	ImageURL      string
	Skills        []string
}
