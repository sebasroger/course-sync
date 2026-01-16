package export

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

// DeleteCourse is the minimal row needed by Eightfold's ef_course_delete XML format.
// Example:
// <EF_Course>
//
//	<title>Managing Performance</title>
//	<lms_course_id>UDM+123</lms_course_id>
//
// </EF_Course>
type DeleteCourse struct {
	Title       string
	LMSCourseID string
}

type efDeleteCourseList struct {
	XMLName xml.Name         `xml:"EF_Course_List"`
	Courses []efDeleteCourse `xml:"EF_Course"`
}

type efDeleteCourse struct {
	Title       string `xml:"title,omitempty"`
	LMSCourseID string `xml:"lms_course_id"`
}

// WriteEFCourseDeleteXML writes an ef_course_delete XML file.
func WriteEFCourseDeleteXML(outPath string, courses []DeleteCourse) error {
	out := efDeleteCourseList{Courses: make([]efDeleteCourse, 0, len(courses))}

	for _, c := range courses {
		id := strings.TrimSpace(c.LMSCourseID)
		if id == "" {
			continue
		}
		out.Courses = append(out.Courses, efDeleteCourse{
			Title:       strings.TrimSpace(c.Title),
			LMSCourseID: id,
		})
	}

	b, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("export: marshal delete xml: %w", err)
	}

	if err := os.WriteFile(outPath, append([]byte(xml.Header), b...), 0o644); err != nil {
		return fmt.Errorf("export: write delete xml: %w", err)
	}
	return nil
}
