package domain

// UnifiedEmployee is the minimal set of fields we need to generate EF employee update XML.
// Eightfold accepts partial updates as long as an identifier is present.
type UnifiedEmployee struct {
	EmployeeID string
	UserID     string
	Level      string
	Emails     []string
}
