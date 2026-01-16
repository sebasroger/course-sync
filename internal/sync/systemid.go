package sync

import "strings"

// BuildSystemID matches the same ID scheme used for Eightfold lms_course_id / systemId.
// IMPORTANT: SourceID must be the raw provider course id (no prefix).
func BuildSystemID(source, sourceID string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "udemy":
		return "UDM+" + strings.TrimSpace(sourceID)
	case "pluralsight":
		return "PLS+" + strings.TrimSpace(sourceID)
	default:
		prefix := strings.ToUpper(strings.TrimSpace(source))
		if prefix == "" {
			prefix = "SRC"
		}
		return prefix + "+" + strings.TrimSpace(sourceID)
	}
}
