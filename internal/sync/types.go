package sync

// EFCourse is the minimal representation we need from Eightfold to compute diffs.
// It is also the schema used for JSON snapshots (eightfold.json).
type EFCourse struct {
	SystemID      string  `json:"systemId"`
	LMSCourseID   string  `json:"lmsCourseId"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	CourseURL     string  `json:"courseUrl"`
	Language      string  `json:"language"`
	Category      string  `json:"category"`
	Difficulty    string  `json:"difficulty"`
	DurationHours float64 `json:"durationHours"`
	Status        string  `json:"status"`
	PublishedDate string  `json:"publishedDate"`
	ImageURL      string  `json:"imageUrl"`
}
